package telemetry

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"tesla-server/internal/fleet"
	"tesla-server/internal/geo"
	"tesla-server/internal/redis"
	"tesla-server/internal/ws"
	"time"

	"github.com/gorilla/websocket"
	"github.com/teslamotors/fleet-telemetry/messages"
	"github.com/teslamotors/fleet-telemetry/protos"
	"google.golang.org/protobuf/proto"
)

//go:embed certs/prod_ca.crt
var defaultProdCA []byte

//go:embed certs/eng_ca.crt
var defaultEngCA []byte

var (
	mediaMu     sync.RWMutex
	latestMedia = make(map[string]*fleet.MediaStateData)
	server      *http.Server
	privateKey  *ecdsa.PrivateKey

	activeConns   = make(map[string]*websocket.Conn)
	activeConnsMu sync.RWMutex
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func InitTelemetryServer(addr string, privKeyPEM []byte, tlsCertFile string, tlsKeyFile string, caCertFile string, useDefaultEngCA bool) error {
	if addr == "" {
		addr = ":8443"
	}

	if len(privKeyPEM) > 0 {
		key, err := parseECPrivateKey(privKeyPEM)
		if err != nil {
			log.Printf("[Telemetry] Warning: failed to parse private key: %v, running without message verification", err)
		} else {
			privateKey = key
			log.Printf("[Telemetry] Private key loaded for message verification")
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/1/vehicles/", handleTelemetry)

	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if tlsCertFile != "" && tlsKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS cert/key: %w", err)
		}

		caCertPool := x509.NewCertPool()

		var defaultCA []byte
		if useDefaultEngCA {
			defaultCA = defaultEngCA
			log.Printf("[Telemetry] Using embedded Tesla engineering CA (eng_ca.crt)")
		} else {
			defaultCA = defaultProdCA
			log.Printf("[Telemetry] Using embedded Tesla production CA (prod_ca.crt)")
		}
		if !caCertPool.AppendCertsFromPEM(defaultCA) {
			return fmt.Errorf("failed to append embedded default CA cert to pool")
		}

		if caCertFile != "" {
			customCaBytes, err := os.ReadFile(caCertFile)
			if err != nil {
				return fmt.Errorf("failed to read custom CA cert: %w", err)
			}
			if !caCertPool.AppendCertsFromPEM(customCaBytes) {
				return fmt.Errorf("failed to append custom CA cert to pool")
			}
			log.Printf("[Telemetry] Appended custom CA cert: %s", caCertFile)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
		}

		server.TLSConfig = tlsConfig

		caMode := "prod"
		if useDefaultEngCA {
			caMode = "eng"
		}
		go func() {
			log.Printf("[Telemetry] mTLS server starting on %s (cert=%s, ca_mode=%s, custom_ca=%s)", addr, tlsCertFile, caMode, caCertFile)
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("[Telemetry] Server error: %v", err)
			}
		}()
	} else {
		go func() {
			log.Printf("[Telemetry] HTTP server starting on %s (no TLS - for development only)", addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Telemetry] Server error: %v", err)
			}
		}()
	}

	return nil
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	vin := pathParts[3]

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		clientType, deviceID, err := messages.CreateIdentityFromCert(r.TLS.PeerCertificates[0])
		if err != nil {
			log.Printf("[Telemetry] Failed to extract identity from client cert: %v", err)
			http.Error(w, "invalid client certificate", http.StatusForbidden)
			return
		}
		if deviceID != vin {
			log.Printf("[Telemetry] VIN mismatch: URL=%s, Cert=%s (clientType=%s)", vin, deviceID, clientType)
			http.Error(w, "VIN mismatch", http.StatusForbidden)
			return
		}
	}

	if isWebSocketRequest(r) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Telemetry] WebSocket upgrade failed for %s: %v", vin, err)
			return
		}
		handleTelemetryWS(vin, conn)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Telemetry] Failed to read body for %s: %v", vin, err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	processRawPayload(vin, body)
	w.WriteHeader(http.StatusOK)
}

func isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

func handleTelemetryWS(vin string, conn *websocket.Conn) {
	activeConnsMu.Lock()
	activeConns[vin] = conn
	activeConnsMu.Unlock()

	redis.SetVehicleStatus(vin, &redis.VehicleStatus{
		Online: true,
		Source: "telemetry",
	})

	defer func() {
		conn.Close()
		activeConnsMu.Lock()
		delete(activeConns, vin)
		activeConnsMu.Unlock()
		log.Printf("[Telemetry] WebSocket disconnected for VIN: %s", vin)
	}()

	log.Printf("[Telemetry] WebSocket connection established for VIN: %s", vin)

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[Telemetry] WebSocket unexpected close for %s: %v", vin, err)
			} else {
				log.Printf("[Telemetry] WebSocket closed for %s: %v", vin, err)
			}
			break
		}

		if messageType == websocket.BinaryMessage {
			go processRawPayload(vin, payload)
		}
	}
}

func processRawPayload(vin string, body []byte) {
	var payload protos.Payload
	if err := proto.Unmarshal(body, &payload); err != nil {
		log.Printf("[Telemetry] Failed to unmarshal protobuf for %s: %v (body_len=%d)", vin, err, len(body))
		handleJSONTelemetry(vin, body)
		return
	}

	processProtobufTelemetry(vin, &payload)
}

func handleJSONTelemetry(vin string, body []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("[Telemetry] Failed to parse JSON fallback for %s: %v", vin, err)
		return
	}

	realtime := &fleet.RealtimeData{
		UpdatedAt: time.Now().UnixMilli(),
	}
	stateFields := map[string]interface{}{}
	media := &fleet.MediaStateData{
		UpdatedAt: time.Now().UnixMilli(),
	}

	hasRealtime := false
	hasState := false
	hasMedia := false

	if v, ok := getFloat(data, "VehicleSpeed"); ok {
		realtime.Speed = v
		hasRealtime = true
	}
	if v, ok := getString(data, "Gear"); ok {
		realtime.Gear = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "Power"); ok {
		realtime.Power = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "PedalPosition"); ok {
		realtime.PedalPosition = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "CruiseSetSpeed"); ok {
		realtime.CruiseSetSpeed = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "LateralAcceleration"); ok {
		realtime.LateralAcceleration = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "LongitudinalAcceleration"); ok {
		realtime.LongitudinalAcceleration = v
		hasRealtime = true
	}
	if loc, ok := data["Location"].(map[string]interface{}); ok {
		var latitude, longitude float64
		if lat, ok := loc["latitude"].(float64); ok {
			latitude = lat
			hasRealtime = true
		}
		if lng, ok := loc["longitude"].(float64); ok {
			longitude = lng
			hasRealtime = true
		}
		// Convert WGS-84 to GCJ-02 for China maps
		lat, lng := geo.WGS84ToGCJ02(latitude, longitude)
		realtime.Latitude = lat
		realtime.Longitude = lng
	}
	if v, ok := getInt(data, "GpsHeading"); ok {
		realtime.Heading = v
		hasRealtime = true
	}
	if v, ok := getInt(data, "GpsState"); ok {
		realtime.GpsState = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "Soc"); ok {
		realtime.Soc = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "BatteryLevel"); ok {
		realtime.BatteryLevel = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "DCChargingPower"); ok {
		realtime.DCChargingPower = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "ACChargingPower"); ok {
		realtime.ACChargingPower = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "PackVoltage"); ok {
		realtime.PackVoltage = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "PackCurrent"); ok {
		realtime.PackCurrent = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "EnergyRemaining"); ok {
		realtime.EnergyRemaining = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "ChargeAmps"); ok {
		realtime.ChargeAmps = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "ChargerVoltage"); ok {
		realtime.ChargerVoltage = v
		hasRealtime = true
	}
	if v, ok := getString(data, "ChargeState"); ok {
		realtime.ChargeState = v
		hasRealtime = true
	}
	if v, ok := getString(data, "DetailedChargeState"); ok {
		realtime.ChargeState = v
		hasRealtime = true
	}
	if v, ok := getBool(data, "FastChargerPresent"); ok {
		realtime.FastChargerPresent = v
		hasRealtime = true
	}

	if v, ok := getBool(data, "Locked"); ok {
		stateFields["locked"] = v
		hasState = true
	}
	if doors, ok := data["DoorState"].(map[string]interface{}); ok {
		doorOpen := false
		if v, ok := doors["DriverFront"].(bool); ok {
			stateFields["door_fl"] = v
			if v { doorOpen = true }
		}
		if v, ok := doors["DriverRear"].(bool); ok {
			stateFields["door_rl"] = v
			if v { doorOpen = true }
		}
		if v, ok := doors["PassengerFront"].(bool); ok {
			stateFields["door_fr"] = v
			if v { doorOpen = true }
		}
		if v, ok := doors["PassengerRear"].(bool); ok {
			stateFields["door_rr"] = v
			if v { doorOpen = true }
		}
		if v, ok := doors["TrunkRear"].(bool); ok {
			stateFields["trunk_open"] = v
		}
		if v, ok := doors["TrunkFront"].(bool); ok {
			stateFields["frunk_open"] = v
		}
		stateFields["door_open"] = doorOpen
		hasState = true
	}
	if v, ok := getBool(data, "SentryMode"); ok {
		stateFields["sentry_mode"] = v
		hasState = true
	}
	if v, ok := getFloat(data, "InsideTemp"); ok {
		stateFields["inside_temp"] = v
		hasState = true
	}
	if v, ok := getFloat(data, "OutsideTemp"); ok {
		stateFields["outside_temp"] = v
		hasState = true
	}
	if v, ok := getBool(data, "ChargePortDoorOpen"); ok {
		stateFields["charge_port_door_open"] = v
		hasState = true
	}
	if v, ok := getInt(data, "ChargeLimitSoc"); ok {
		stateFields["charge_limit_soc"] = v
		hasState = true
	}
	if v, ok := getFloat(data, "TpmsPressureFl"); ok {
		stateFields["tpms_fl"] = v
		hasState = true
	}
	if v, ok := getFloat(data, "TpmsPressureFr"); ok {
		stateFields["tpms_fr"] = v
		hasState = true
	}
	if v, ok := getFloat(data, "TpmsPressureRl"); ok {
		stateFields["tpms_rl"] = v
		hasState = true
	}
	if v, ok := getFloat(data, "TpmsPressureRr"); ok {
		stateFields["tpms_rr"] = v
		hasState = true
	}

	// Handle additional state fields from nested JSON format {"key": {"value": actual_value}}
	for key, rawVal := range data {
		v, ok := rawVal.(map[string]interface{})
		if !ok {
			continue
		}
		switch key {
		// Vehicle state fields
		case "Odometer":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["odometer_km"] = milesToKm(val)
			}
			hasState = true
		case "CenterDisplay":
			stateFields["center_display_state"] = getIntVal(v, "value")
			hasState = true
		case "BrakePedal":
			stateFields["brake_pedal"] = getBoolVal(v, "value")
			hasState = true
		case "DriveRail":
			stateFields["drive_rail"] = getBoolVal(v, "value")
			hasState = true
		case "FdWindow":
			stateFields["fd_window"] = getBoolVal(v, "value")
			hasState = true
		case "FpWindow":
			stateFields["fp_window"] = getBoolVal(v, "value")
			hasState = true
		case "DriverSeatBelt":
			stateFields["driver_seat_belt"] = getBoolVal(v, "value")
			hasState = true
		case "DriverSeatOccupied":
			stateFields["driver_seat_occupied"] = getBoolVal(v, "value")
			hasState = true
		case "GuestModeEnabled":
			stateFields["guest_mode_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "HomelinkNearby":
			stateFields["homelink_nearby"] = getBoolVal(v, "value")
			hasState = true
		case "HomelinkDeviceCount":
			stateFields["homelink_device_count"] = getIntVal(v, "value")
			hasState = true
		case "ServiceMode":
			stateFields["service_mode"] = getBoolVal(v, "value")
			hasState = true
		case "Version":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["version"] = s
			}
			hasState = true
		case "CarType":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["car_type"] = s
			}
			hasState = true
		case "ExteriorColor":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["exterior_color"] = s
			}
			hasState = true
		case "EfficiencyPackage":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["efficiency_package"] = s
			}
			hasState = true

		// Climate/HVAC fields
		case "HvacPower":
			powerVal := getFloatVal(v, "value")
			stateFields["hvac_power"] = powerVal
			stateFields["is_ac_on"] = powerVal > 0
			stateFields["is_climate_on"] = powerVal > 0
			hasState = true
		case "HvacLeftTemperatureRequest":
			stateFields["driver_temp_setting"] = getFloatVal(v, "value")
			hasState = true
		case "HvacRightTemperatureRequest":
			stateFields["passenger_temp_setting"] = getFloatVal(v, "value")
			hasState = true
		case "DefrostMode":
			stateFields["defrost_mode"] = getIntVal(v, "value")
			hasState = true
		case "HvacACEnabled":
			stateFields["hvac_ac_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "HvacSteeringWheelHeatLevel":
			level := getIntVal(v, "value")
			stateFields["steering_wheel_heater"] = level > 0
			stateFields["hvac_steering_wheel_heat_level"] = level
			hasState = true
		case "HvacSteeringWheelHeatAuto":
			stateFields["hvac_steering_wheel_heat_auto"] = getBoolVal(v, "value")
			hasState = true
		case "HvacFanSpeed":
			stateFields["hvac_fan_speed"] = getIntVal(v, "value")
			hasState = true
		case "HvacAutoMode":
			stateFields["hvac_auto_mode"] = getBoolVal(v, "value")
			hasState = true
		case "ClimateKeeperMode":
			stateFields["climate_keeper_mode"] = getIntVal(v, "value")
			hasState = true
		case "DefrostForPreconditioning":
			stateFields["defrost_for_preconditioning"] = getBoolVal(v, "value")
			hasState = true
		case "AutoSeatClimateLeft":
			stateFields["auto_seat_climate_left"] = getBoolVal(v, "value")
			hasState = true
		case "AutoSeatClimateRight":
			stateFields["auto_seat_climate_right"] = getBoolVal(v, "value")
			hasState = true
		case "ClimateSeatCoolingFrontLeft":
			stateFields["climate_seat_cooling_front_left"] = getIntVal(v, "value")
			hasState = true
		case "ClimateSeatCoolingFrontRight":
			stateFields["climate_seat_cooling_front_right"] = getIntVal(v, "value")
			hasState = true
		case "CabinOverheatProtectionMode":
			stateFields["cabin_overheat_protection_mode"] = getIntVal(v, "value")
			hasState = true
		case "CabinOverheatProtectionTemperatureLimit":
			stateFields["cabin_overheat_protection_temperature_limit"] = getFloatVal(v, "value")
			hasState = true
		case "SeatHeaterLeft":
			stateFields["seat_heater_left"] = getIntVal(v, "value")
			hasState = true
		case "SeatHeaterRight":
			stateFields["seat_heater_right"] = getIntVal(v, "value")
			hasState = true
		case "SeatHeaterRearLeft":
			stateFields["seat_heater_rear_left"] = getIntVal(v, "value")
			hasState = true
		case "SeatHeaterRearRight":
			stateFields["seat_heater_rear_right"] = getIntVal(v, "value")
			hasState = true
		case "SeatHeaterRearCenter":
			stateFields["seat_heater_rear_center"] = getIntVal(v, "value")
			hasState = true
		case "WiperHeatEnabled":
			stateFields["wiper_heat_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "RearDisplayHvacEnabled":
			stateFields["rear_display_hvac_enabled"] = getBoolVal(v, "value")
			hasState = true

		// Charging detail fields
		case "ChargeRateMilePerHour":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["charge_speed"] = milesToKm(val)
			}
			hasState = true
		case "IdealBatteryRange":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["range_km"] = milesToKm(val)
			}
			hasState = true
		case "EstBatteryRange":
			if val := getFloatVal(v, "value"); val > 0 {
				if _, ok := stateFields["range_km"]; !ok {
					stateFields["range_km"] = milesToKm(val)
				}
			}
			hasState = true
		case "BatteryHeaterOn":
			stateFields["battery_heater_on"] = getBoolVal(v, "value")
			hasState = true
		case "DCChargingEnergyIn":
			stateFields["dc_charging_energy_in"] = getFloatVal(v, "value")
			hasState = true
		case "ACChargingEnergyIn":
			stateFields["ac_charging_energy_in"] = getFloatVal(v, "value")
			hasState = true
		case "EstimatedHoursToChargeTermination":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["minutes_to_full"] = int(val * 60)
			}
			hasState = true
		case "FastChargerType":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["fast_charger_type"] = s
			}
			hasState = true
		case "ChargePortLatch":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["charge_port_latch"] = s
			}
			hasState = true
		case "ChargingCableType":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["charging_cable_type"] = s
			}
			hasState = true
		case "ChargeEnableRequest":
			stateFields["charge_enable_request"] = getBoolVal(v, "value")
			hasState = true
		case "ChargeCurrentRequest":
			stateFields["charge_current_request"] = getIntVal(v, "value")
			hasState = true
		case "ChargeCurrentRequestMax":
			stateFields["charge_current_request_max"] = getIntVal(v, "value")
			hasState = true
		case "ChargerPhases":
			stateFields["charger_phases"] = getIntVal(v, "value")
			hasState = true
		case "ChargePortColdWeatherMode":
			stateFields["charge_port_cold_weather_mode"] = getBoolVal(v, "value")
			hasState = true
		case "BMSState":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["bms_state"] = s
			}
			hasState = true
		case "BmsFullchargecomplete":
			stateFields["bms_full_charge_complete"] = getBoolVal(v, "value")
			hasState = true
		case "LifetimeEnergyUsed":
			stateFields["lifetime_energy_used"] = getFloatVal(v, "value")
			hasState = true
		case "DCDCEnable":
			stateFields["dcdc_enable"] = getBoolVal(v, "value")
			hasState = true
		case "BrickVoltageMax":
			stateFields["brick_voltage_max"] = getFloatVal(v, "value")
			hasState = true
		case "BrickVoltageMin":
			stateFields["brick_voltage_min"] = getFloatVal(v, "value")
			hasState = true
		case "IsolationResistance":
			stateFields["isolation_resistance"] = getFloatVal(v, "value")
			hasState = true
		case "PreconditioningEnabled":
			stateFields["preconditioning_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "NotEnoughPowerToHeat":
			stateFields["not_enough_power_to_heat"] = getBoolVal(v, "value")
			hasState = true
		case "SuperchargerSessionTripPlanner":
			stateFields["supercharger_session_trip_planner"] = getBoolVal(v, "value")
			hasState = true
		case "ModuleTempMax":
			stateFields["module_temp_max"] = getFloatVal(v, "value")
			hasState = true
		case "ModuleTempMin":
			stateFields["module_temp_min"] = getFloatVal(v, "value")
			hasState = true
		case "NumModuleTempMax":
			stateFields["num_module_temp_max"] = getIntVal(v, "value")
			hasState = true
		case "NumModuleTempMin":
			stateFields["num_module_temp_min"] = getIntVal(v, "value")
			hasState = true
		case "NumBrickVoltageMax":
			stateFields["num_brick_voltage_max"] = getIntVal(v, "value")
			hasState = true
		case "NumBrickVoltageMin":
			stateFields["num_brick_voltage_min"] = getIntVal(v, "value")
			hasState = true

		// Safety/Lights fields
		case "CurrentLimitMph":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["current_limit_mph"] = val
			}
			hasState = true
		case "CruiseFollowDistance":
			stateFields["cruise_follow_distance"] = getIntVal(v, "value")
			hasState = true
		case "AutomaticBlindSpotCamera":
			stateFields["automatic_blind_spot_camera"] = getBoolVal(v, "value")
			hasState = true
		case "BlindSpotCollisionWarningChime":
			stateFields["blind_spot_collision_warning_chime"] = getBoolVal(v, "value")
			hasState = true
		case "ForwardCollisionWarning":
			stateFields["forward_collision_warning"] = getBoolVal(v, "value")
			hasState = true
		case "LaneDepartureAvoidance":
			stateFields["lane_departure_avoidance"] = getBoolVal(v, "value")
			hasState = true
		case "EmergencyLaneDepartureAvoidance":
			stateFields["emergency_lane_departure_avoidance"] = getBoolVal(v, "value")
			hasState = true
		case "AutomaticEmergencyBrakingOff":
			stateFields["automatic_emergency_braking_off"] = getBoolVal(v, "value")
			hasState = true
		case "LightsHazardsActive":
			stateFields["lights_hazards_active"] = getBoolVal(v, "value")
			hasState = true
		case "LightsHighBeams":
			stateFields["lights_high_beams"] = getBoolVal(v, "value")
			hasState = true
		case "LightsTurnSignal":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["lights_turn_signal"] = s
			}
			hasState = true

		// Navigation fields
		case "DestinationLocation":
			if loc, ok := v["value"].(map[string]interface{}); ok {
				if lat, ok := loc["latitude"].(float64); ok {
					stateFields["destination_latitude"] = lat
				}
				if lng, ok := loc["longitude"].(float64); ok {
					stateFields["destination_longitude"] = lng
				}
			}
			hasState = true
		case "DestinationName":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["destination_name"] = s
			}
			hasState = true

		// Powershare fields
		case "PowershareStatus":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["powershare_status"] = s
			}
			hasState = true
		case "PowershareType":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["powershare_type"] = s
			}
			hasState = true
		case "PowershareInstantaneousPowerKW":
			stateFields["powershare_instantaneous_power_kw"] = getFloatVal(v, "value")
			hasState = true
		case "PowershareHoursLeft":
			stateFields["powershare_hours_left"] = getFloatVal(v, "value")
			hasState = true
		case "PowershareStopReason":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["powershare_stop_reason"] = s
			}
			hasState = true

		// Software update fields
		case "SoftwareUpdateVersion":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["software_update_version"] = s
			}
			hasState = true
		case "SoftwareUpdateDownloadPercentComplete":
			stateFields["software_update_download_percent"] = getIntVal(v, "value")
			hasState = true
		case "SoftwareUpdateExpectedDurationMinutes":
			stateFields["software_update_expected_duration_minutes"] = getIntVal(v, "value")
			hasState = true
		case "SoftwareUpdateInstallationPercentComplete":
			stateFields["software_update_installation_percent"] = getIntVal(v, "value")
			hasState = true
		case "SoftwareUpdateScheduledStartTime":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["software_update_scheduled_start_time"] = s
			}
			hasState = true

		// Vehicle config fields
		case "VehicleName":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["vehicle_name"] = s
			}
			hasState = true
		case "Trim":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["trim"] = s
			}
			hasState = true
		case "RoofColor":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["roof_color"] = s
			}
			hasState = true
		case "WheelType":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["wheel_type"] = s
			}
			hasState = true
		case "EuropeVehicle":
			stateFields["europe_vehicle"] = getBoolVal(v, "value")
			hasState = true
		case "RightHandDrive":
			stateFields["right_hand_drive"] = getBoolVal(v, "value")
			hasState = true
		case "RearSeatHeaters":
			stateFields["rear_seat_heaters"] = getIntVal(v, "value")
			hasState = true
		case "SunroofInstalled":
			stateFields["sunroof_installed"] = getBoolVal(v, "value")
			hasState = true
		case "RemoteStartEnabled":
			stateFields["remote_start_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "Setting24HourTime":
			stateFields["setting_24_hour_time"] = getBoolVal(v, "value")
			hasState = true
		case "SettingChargeUnit":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["setting_charge_unit"] = s
			}
			hasState = true
		case "SettingDistanceUnit":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["setting_distance_unit"] = s
			}
			hasState = true
		case "SettingTemperatureUnit":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["setting_temperature_unit"] = s
			}
			hasState = true
		case "SettingTirePressureUnit":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["setting_tire_pressure_unit"] = s
			}
			hasState = true

		// Cybertruck fields
		case "TonneauPosition":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["tonneau_position"] = s
			}
			hasState = true
		case "TonneauOpenPercent":
			stateFields["tonneau_open_percent"] = getIntVal(v, "value")
			hasState = true
		case "TonneauTentMode":
			stateFields["tonneau_tent_mode"] = getBoolVal(v, "value")
			hasState = true
		case "OffroadLightbarPresent":
			stateFields["offroad_lightbar_present"] = getBoolVal(v, "value")
			hasState = true

		// TPMS detail fields
		case "TpmsLastSeenPressureTimeFl":
			stateFields["tpms_last_seen_pressure_time_fl"] = getIntVal(v, "value")
			hasState = true
		case "TpmsLastSeenPressureTimeFr":
			stateFields["tpms_last_seen_pressure_time_fr"] = getIntVal(v, "value")
			hasState = true
		case "TpmsLastSeenPressureTimeRl":
			stateFields["tpms_last_seen_pressure_time_rl"] = getIntVal(v, "value")
			hasState = true
		case "TpmsLastSeenPressureTimeRr":
			stateFields["tpms_last_seen_pressure_time_rr"] = getIntVal(v, "value")
			hasState = true
		case "TpmsSoftWarnings":
			stateFields["tpms_soft_warnings"] = getBoolVal(v, "value")
			hasState = true
		case "TpmsHardWarnings":
			stateFields["tpms_hard_warnings"] = getBoolVal(v, "value")
			hasState = true

		// Additional fields
		case "PinToDriveEnabled":
			stateFields["pin_to_drive_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "PairedPhoneKeyAndKeyFobQty":
			stateFields["paired_phone_key_and_key_fob_qty"] = getIntVal(v, "value")
			hasState = true
		case "PassengerSeatBelt":
			stateFields["passenger_seat_belt"] = getBoolVal(v, "value")
			hasState = true
		case "MilesSinceReset":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["miles_since_reset"] = val
			}
			hasState = true
		case "SelfDrivingMilesSinceReset":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["self_driving_miles_since_reset"] = val
			}
			hasState = true
		case "LocatedAtHome":
			stateFields["located_at_home"] = getBoolVal(v, "value")
			hasState = true
		case "LocatedAtWork":
			stateFields["located_at_work"] = getBoolVal(v, "value")
			hasState = true
		case "LocatedAtFavorite":
			stateFields["located_at_favorite"] = getBoolVal(v, "value")
			hasState = true
		case "RouteLastUpdated":
			if s := getStringVal(v, "value"); s != "" {
				stateFields["route_last_updated"] = s
			}
			hasState = true
		case "RouteTrafficMinutesDelay":
			stateFields["route_traffic_minutes_delay"] = getFloatVal(v, "value")
			hasState = true
		case "MilesToArrival":
			if val := getFloatVal(v, "value"); val > 0 {
				stateFields["miles_to_arrival"] = val
			}
			hasState = true
		case "MinutesToArrival":
			stateFields["minutes_to_arrival"] = getFloatVal(v, "value")
			hasState = true
		case "ExpectedEnergyPercentAtTripArrival":
			stateFields["expected_energy_percent_at_trip_arrival"] = getFloatVal(v, "value")
			hasState = true
		case "GpsState":
			if val, ok := getInt(v, "value"); ok {
				realtime.GpsState = val
				hasRealtime = true
			}
		case "SeatVentEnabled":
			stateFields["seat_vent_enabled"] = getBoolVal(v, "value")
			hasState = true
		case "Hvil":
			stateFields["hvil"] = getBoolVal(v, "value")
			hasState = true
		case "MediaAudioVolumeIncrement":
			stateFields["media_audio_volume_increment"] = getFloatVal(v, "value")
			hasState = true
		case "MediaAudioVolumeMax":
			stateFields["media_audio_volume_max"] = getFloatVal(v, "value")
			hasState = true
		case "DetailedChargeState":
			if s := getStringVal(v, "value"); s != "" {
				realtime.ChargeState = s
				hasRealtime = true
			}
		default:
			snakeKey := toSnakeCase(key)
			if val, ok := v["value"]; ok {
				switch tv := val.(type) {
				case bool:
					stateFields[snakeKey] = tv
				case float64:
					stateFields[snakeKey] = tv
				case string:
					stateFields[snakeKey] = tv
				case int:
					stateFields[snakeKey] = tv
				case map[string]interface{}:
					// Skip complex objects
				}
				hasState = true
			}
		}
	}

	if v, ok := getString(data, "MediaPlaybackStatus"); ok {
		media.PlaybackStatus = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaPlaybackSource"); ok {
		media.AudioSource = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaAudioSource"); ok {
		media.AudioSource = v
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaAudioVolume"); ok {
		media.Volume = int(v)
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaVolume"); ok {
		media.Volume = int(v)
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingTitle"); ok {
		media.NowPlayingTitle = v
		hasMedia = true
	}
	if v, ok := getString(data, "NowPlayingTitle"); ok {
		media.NowPlayingTitle = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingArtist"); ok {
		media.NowPlayingArtist = v
		hasMedia = true
	}
	if v, ok := getString(data, "NowPlayingArtist"); ok {
		media.NowPlayingArtist = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingAlbum"); ok {
		media.NowPlayingAlbum = v
		hasMedia = true
	}
	if v, ok := getString(data, "NowPlayingAlbum"); ok {
		media.NowPlayingAlbum = v
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaNowPlayingDuration"); ok {
		media.NowPlayingDuration = int(v)
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaNowPlayingElapsed"); ok {
		media.NowPlayingElapsed = int(v)
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingStation"); ok {
		media.NowPlayingStation = v
		hasMedia = true
	}

	if hasRealtime {
		updateRealtimeState(vin, realtime)
	}
	if hasState {
		updateVehicleStateFields(vin, stateFields)
	}
	if hasMedia {
		updateMediaState(vin, media)
	}
}

func processProtobufTelemetry(vin string, payload *protos.Payload) {
	realtime := &fleet.RealtimeData{
		UpdatedAt: time.Now().UnixMilli(),
	}
	stateFields := map[string]interface{}{}
	media := &fleet.MediaStateData{
		UpdatedAt: time.Now().UnixMilli(),
	}

	hasRealtime := false
	hasState := false
	hasMedia := false

	for _, datum := range payload.Data {
		if datum.Value == nil || datum.Value.GetInvalid() {
			continue
		}

		switch datum.Key {
		case protos.Field_VehicleSpeed:
			realtime.Speed = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_Gear:
			switch datum.Value.GetShiftStateValue() {
			case protos.ShiftState_ShiftStateP:
				realtime.Gear = "P"
			case protos.ShiftState_ShiftStateR:
				realtime.Gear = "R"
			case protos.ShiftState_ShiftStateN:
				realtime.Gear = "N"
			case protos.ShiftState_ShiftStateD:
				realtime.Gear = "D"
			}
			hasRealtime = true
		case protos.Field_PedalPosition:
			realtime.PedalPosition = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_CruiseSetSpeed:
			realtime.CruiseSetSpeed = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_LateralAcceleration:
			realtime.LateralAcceleration = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_LongitudinalAcceleration:
			realtime.LongitudinalAcceleration = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_Location:
			loc := datum.Value.GetLocationValue()
			if loc != nil {
				// Convert WGS-84 to GCJ-02 for China maps
				lat, lng := geo.WGS84ToGCJ02(loc.GetLatitude(), loc.GetLongitude())
				realtime.Latitude = lat
				realtime.Longitude = lng
				hasRealtime = true
			}
		case protos.Field_GpsHeading:
			realtime.Heading = getIntValue(datum.Value)
			hasRealtime = true
		case protos.Field_GpsState:
			realtime.GpsState = getIntValue(datum.Value)
			hasRealtime = true
		case protos.Field_Soc:
			realtime.Soc = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_BatteryLevel:
			realtime.BatteryLevel = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_DCChargingPower:
			realtime.DCChargingPower = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_ACChargingPower:
			realtime.ACChargingPower = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_PackVoltage:
			realtime.PackVoltage = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_PackCurrent:
			realtime.PackCurrent = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_EnergyRemaining:
			realtime.EnergyRemaining = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_ChargeAmps:
			realtime.ChargeAmps = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_ChargerVoltage:
			realtime.ChargerVoltage = getFloat64Value(datum.Value)
			hasRealtime = true
		case protos.Field_ChargeState:
			realtime.ChargeState = chargingStateToString(datum.Value.GetChargingValue())
			hasRealtime = true
		case protos.Field_DetailedChargeState:
			realtime.ChargeState = detailedChargeStateToString(datum.Value.GetDetailedChargeStateValue())
			hasRealtime = true
		case protos.Field_FastChargerPresent:
			realtime.FastChargerPresent = datum.Value.GetBooleanValue()
			hasRealtime = true

		case protos.Field_Locked:
			stateFields["locked"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DoorState:
			doors := datum.Value.GetDoorValue()
			if doors != nil {
				doorOpen := false
				if doors.GetDriverFront() {
					stateFields["door_fl"] = true
					doorOpen = true
				}
				if doors.GetDriverRear() {
					stateFields["door_rl"] = true
					doorOpen = true
				}
				if doors.GetPassengerFront() {
					stateFields["door_fr"] = true
					doorOpen = true
				}
				if doors.GetPassengerRear() {
					stateFields["door_rr"] = true
					doorOpen = true
				}
				if doors.GetTrunkRear() {
					stateFields["trunk_open"] = true
				}
				if doors.GetTrunkFront() {
					stateFields["frunk_open"] = true
				}
				stateFields["door_open"] = doorOpen
				hasState = true
			}
		case protos.Field_SentryMode:
			sentryState := datum.Value.GetSentryModeStateValue()
			stateFields["sentry_mode"] = sentryState != protos.SentryModeState_SentryModeStateOff && sentryState != protos.SentryModeState_SentryModeStateUnknown
			hasState = true
		case protos.Field_InsideTemp:
			stateFields["inside_temp"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_OutsideTemp:
			stateFields["outside_temp"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_ChargePortDoorOpen:
			stateFields["charge_port_door_open"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ChargeLimitSoc:
			stateFields["charge_limit_soc"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_TpmsPressureFl:
			stateFields["tpms_fl"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_TpmsPressureFr:
			stateFields["tpms_fr"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_TpmsPressureRl:
			stateFields["tpms_rl"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_TpmsPressureRr:
			stateFields["tpms_rr"] = getFloat64Value(datum.Value)
			hasState = true

		case protos.Field_Odometer:
			stateFields["odometer_km"] = getFloat64Value(datum.Value) * 1.60934
			hasState = true
		case protos.Field_CenterDisplay:
			stateFields["center_display_state"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_HvacPower:
			hvacOn := datum.Value.GetBooleanValue()
			stateFields["hvac_power"] = hvacOn
			stateFields["is_ac_on"] = hvacOn
			stateFields["is_climate_on"] = hvacOn
			hasState = true
		case protos.Field_HvacLeftTemperatureRequest:
			stateFields["driver_temp_setting"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_HvacRightTemperatureRequest:
			stateFields["passenger_temp_setting"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_DefrostMode:
			stateFields["defrost_mode"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargeRateMilePerHour:
			stateFields["charge_speed"] = getFloat64Value(datum.Value) * 1.60934
			hasState = true
		case protos.Field_IdealBatteryRange:
			stateFields["range_km"] = getFloat64Value(datum.Value) * 1.60934
			hasState = true
		case protos.Field_EstBatteryRange:
			r := getFloat64Value(datum.Value) * 1.60934
			if stateFields["range_km"] == nil {
				stateFields["range_km"] = r
			}
			hasState = true
		case protos.Field_BrakePedal:
			stateFields["brake_pedal"] = datum.Value.GetBooleanValue()
			hasState = true

		case protos.Field_DriveRail:
			stateFields["drive_rail"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_SeatHeaterLeft:
			stateFields["seat_heater_left"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_SeatHeaterRight:
			stateFields["seat_heater_right"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_SeatHeaterRearLeft:
			stateFields["seat_heater_rear_left"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_SeatHeaterRearRight:
			stateFields["seat_heater_rear_right"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_DriverSeatBelt:
			stateFields["driver_seat_belt"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DriverSeatOccupied:
			stateFields["driver_seat_occupied"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_FdWindow:
			stateFields["fd_window"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_FpWindow:
			stateFields["fp_window"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_HvacACEnabled:
			stateFields["hvac_ac_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HvacSteeringWheelHeatLevel:
			stateFields["steering_wheel_heater"] = getIntValue(datum.Value) > 0
			stateFields["hvac_steering_wheel_heat_level"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_HvacSteeringWheelHeatAuto:
			stateFields["hvac_steering_wheel_heat_auto"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HvacFanSpeed:
			stateFields["hvac_fan_speed"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_HvacAutoMode:
			stateFields["hvac_auto_mode"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ClimateKeeperMode:
			stateFields["climate_keeper_mode"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_DefrostForPreconditioning:
			stateFields["defrost_for_preconditioning"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_AutoSeatClimateLeft:
			stateFields["auto_seat_climate_left"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_AutoSeatClimateRight:
			stateFields["auto_seat_climate_right"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ClimateSeatCoolingFrontLeft:
			stateFields["climate_seat_cooling_front_left"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ClimateSeatCoolingFrontRight:
			stateFields["climate_seat_cooling_front_right"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_CabinOverheatProtectionMode:
			stateFields["cabin_overheat_protection_mode"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_CabinOverheatProtectionTemperatureLimit:
			stateFields["cabin_overheat_protection_temperature_limit"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_BatteryHeaterOn:
			stateFields["battery_heater_on"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DCChargingEnergyIn:
			stateFields["dc_charging_energy_in"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_ACChargingEnergyIn:
			stateFields["ac_charging_energy_in"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_EstimatedHoursToChargeTermination:
			stateFields["minutes_to_full"] = getFloat64Value(datum.Value) * 60
			hasState = true
		case protos.Field_FastChargerType:
			stateFields["fast_charger_type"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargePortLatch:
			stateFields["charge_port_latch"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargingCableType:
			stateFields["charging_cable_type"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargeEnableRequest:
			stateFields["charge_enable_request"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ChargeCurrentRequest:
			stateFields["charge_current_request"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargeCurrentRequestMax:
			stateFields["charge_current_request_max"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargerPhases:
			stateFields["charger_phases"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_ChargePortColdWeatherMode:
			stateFields["charge_port_cold_weather_mode"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DestinationLocation:
			loc := datum.Value.GetLocationValue()
			if loc != nil {
				stateFields["destination_latitude"] = loc.GetLatitude()
				stateFields["destination_longitude"] = loc.GetLongitude()
			}
			hasState = true
		case protos.Field_DestinationName:
			stateFields["destination_name"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_CarType:
			stateFields["car_type"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_ExteriorColor:
			stateFields["exterior_color"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_EfficiencyPackage:
			stateFields["efficiency_package"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_Version:
			stateFields["version"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_GuestModeEnabled:
			stateFields["guest_mode_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HomelinkNearby:
			stateFields["homelink_nearby"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HomelinkDeviceCount:
			stateFields["homelink_device_count"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_LightsHazardsActive:
			stateFields["lights_hazards_active"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LightsHighBeams:
			stateFields["lights_high_beams"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LightsTurnSignal:
			ts := datum.Value.GetTurnSignalStateValue()
			switch ts {
			case protos.TurnSignalState_TurnSignalStateLeft:
				stateFields["lights_turn_signal"] = "left"
			case protos.TurnSignalState_TurnSignalStateRight:
				stateFields["lights_turn_signal"] = "right"
			default:
				stateFields["lights_turn_signal"] = "off"
			}
			hasState = true
		case protos.Field_ServiceMode:
			stateFields["service_mode"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_BMSState:
			stateFields["bms_state"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_BmsFullchargecomplete:
			stateFields["bms_full_charge_complete"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LifetimeEnergyUsed:
			stateFields["lifetime_energy_used"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_CurrentLimitMph:
			stateFields["current_limit_mph"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_CruiseFollowDistance:
			stateFields["cruise_follow_distance"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_AutomaticBlindSpotCamera:
			stateFields["automatic_blind_spot_camera"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_BlindSpotCollisionWarningChime:
			stateFields["blind_spot_collision_warning_chime"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ForwardCollisionWarning:
			stateFields["forward_collision_warning"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_LaneDepartureAvoidance:
			stateFields["lane_departure_avoidance"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_EmergencyLaneDepartureAvoidance:
			stateFields["emergency_lane_departure_avoidance"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_AutomaticEmergencyBrakingOff:
			stateFields["automatic_emergency_braking_off"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DCDCEnable:
			stateFields["dcdc_enable"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_BrickVoltageMax:
			stateFields["brick_voltage_max"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_BrickVoltageMin:
			stateFields["brick_voltage_min"] = getIntValue(datum.Value)
			hasState = true
		case protos.Field_IsolationResistance:
			stateFields["isolation_resistance"] = getFloat64Value(datum.Value)
			hasState = true
		case protos.Field_Hvil:
			stateFields["hvil"] = getIntValue(datum.Value)
			hasState = true

		case protos.Field_MediaPlaybackStatus:
			media.PlaybackStatus = mediaStatusToString(datum.Value.GetMediaStatusValue())
			hasMedia = true
		case protos.Field_MediaPlaybackSource:
			media.AudioSource = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaAudioVolume:
			media.Volume = getIntValue(datum.Value)
			hasMedia = true
		case protos.Field_MediaNowPlayingDuration:
			media.NowPlayingDuration = getIntValue(datum.Value)
			hasMedia = true
		case protos.Field_MediaNowPlayingElapsed:
			media.NowPlayingElapsed = getIntValue(datum.Value)
			hasMedia = true
		case protos.Field_MediaNowPlayingArtist:
			media.NowPlayingArtist = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaNowPlayingTitle:
			media.NowPlayingTitle = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaNowPlayingAlbum:
			media.NowPlayingAlbum = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaNowPlayingStation:
			media.NowPlayingStation = datum.Value.GetStringValue()
			hasMedia = true
		default:
			fieldName := strings.TrimPrefix(datum.Key.String(), "Field_")
			snake := toSnakeCase(fieldName)
			switch v := datum.Value.GetValue().(type) {
			case *protos.Value_BooleanValue:
				stateFields[snake] = v.BooleanValue
			case *protos.Value_DoubleValue:
				stateFields[snake] = v.DoubleValue
			case *protos.Value_FloatValue:
				stateFields[snake] = v.FloatValue
			case *protos.Value_IntValue:
				stateFields[snake] = v.IntValue
			case *protos.Value_LongValue:
				stateFields[snake] = v.LongValue
			case *protos.Value_StringValue:
				stateFields[snake] = v.StringValue
			default:
				stateFields[snake] = fmt.Sprintf("%v", datum.Value)
			}
			hasState = true
		}
	}

	if hasRealtime {
		updateRealtimeState(vin, realtime)
	}
	if hasState {
		updateVehicleStateFields(vin, stateFields)
	}
	if hasMedia {
		updateMediaState(vin, media)
	}
}

func getFloat64Value(v *protos.Value) float64 {
	switch v.GetValue().(type) {
	case *protos.Value_DoubleValue:
		return v.GetDoubleValue()
	case *protos.Value_FloatValue:
		return float64(v.GetFloatValue())
	case *protos.Value_IntValue:
		return float64(v.GetIntValue())
	case *protos.Value_LongValue:
		return float64(v.GetLongValue())
	default:
		return 0
	}
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func getIntValue(v *protos.Value) int {
	switch v.GetValue().(type) {
	case *protos.Value_IntValue:
		return int(v.GetIntValue())
	case *protos.Value_LongValue:
		return int(v.GetLongValue())
	case *protos.Value_DoubleValue:
		return int(v.GetDoubleValue())
	case *protos.Value_FloatValue:
		return int(v.GetFloatValue())
	default:
		return 0
	}
}

func chargingStateToString(state protos.ChargingState) string {
	switch state {
	case protos.ChargingState_ChargeStateDisconnected:
		return "Disconnected"
	case protos.ChargingState_ChargeStateNoPower:
		return "NoPower"
	case protos.ChargingState_ChargeStateStarting:
		return "Starting"
	case protos.ChargingState_ChargeStateCharging:
		return "Charging"
	case protos.ChargingState_ChargeStateComplete:
		return "Complete"
	case protos.ChargingState_ChargeStateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

func detailedChargeStateToString(state protos.DetailedChargeStateValue) string {
	switch state {
	case protos.DetailedChargeStateValue_DetailedChargeStateDisconnected:
		return "Disconnected"
	case protos.DetailedChargeStateValue_DetailedChargeStateNoPower:
		return "NoPower"
	case protos.DetailedChargeStateValue_DetailedChargeStateStarting:
		return "Starting"
	case protos.DetailedChargeStateValue_DetailedChargeStateCharging:
		return "Charging"
	case protos.DetailedChargeStateValue_DetailedChargeStateComplete:
		return "Complete"
	case protos.DetailedChargeStateValue_DetailedChargeStateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

func mediaStatusToString(status protos.MediaStatus) string {
	switch status {
	case protos.MediaStatus_MediaStatusStopped:
		return "Stopped"
	case protos.MediaStatus_MediaStatusPlaying:
		return "Playing"
	case protos.MediaStatus_MediaStatusPaused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func updateRealtimeState(vin string, realtime *fleet.RealtimeData) {
	if err := redis.SetVehicleRealtime(vin, realtime); err != nil {
		log.Printf("[Telemetry] Failed to update Redis realtime for %s: %v", vin, err)
	}

	redis.SetVehicleStatus(vin, &redis.VehicleStatus{
		Online: true,
		Source: "telemetry",
	})

	ws.BroadcastRealtimeUpdate(vin, realtime)

	log.Printf("[Telemetry] Realtime update for %s: speed=%.1f gear=%s soc=%.0f power=%.1f lat=%.4f lng=%.4f",
		vin, realtime.Speed, realtime.Gear, realtime.Soc, realtime.Power, realtime.Latitude, realtime.Longitude)
}

func updateVehicleStateFields(vin string, fields map[string]interface{}) {
	if err := redis.UpdateVehicleStateFields(vin, fields); err != nil {
		log.Printf("[Telemetry] Failed to update Redis state for %s: %v", vin, err)
	}

	ws.BroadcastStateUpdate(vin, fields)

	log.Printf("[Telemetry] State update for %s: %d fields", vin, len(fields))
}

func updateMediaState(vin string, media *fleet.MediaStateData) {
	mediaMu.Lock()
	latestMedia[vin] = media
	mediaMu.Unlock()

	fields := map[string]interface{}{
		"media_playback_status": media.PlaybackStatus,
		"media_audio_source":    media.AudioSource,
		"media_volume":          media.Volume,
		"now_playing_title":     media.NowPlayingTitle,
		"now_playing_artist":    media.NowPlayingArtist,
		"now_playing_album":     media.NowPlayingAlbum,
		"now_playing_duration":  media.NowPlayingDuration,
		"now_playing_elapsed":   media.NowPlayingElapsed,
		"now_playing_station":   media.NowPlayingStation,
	}

	if err := redis.UpdateVehicleStateFields(vin, fields); err != nil {
		log.Printf("[Telemetry] Failed to update Redis for %s: %v", vin, err)
	}

	if ws.DefaultHub != nil {
		ws.DefaultHub.BroadcastToVIN(vin, "media_state", fields)
	}

	log.Printf("[Telemetry] Media update for %s: status=%q source=%q title=%q artist=%q volume=%d",
		vin, media.PlaybackStatus, media.AudioSource, media.NowPlayingTitle, media.NowPlayingArtist, media.Volume)
}

func GetLatestMedia(vin string) *fleet.MediaStateData {
	mediaMu.RLock()
	defer mediaMu.RUnlock()
	m, ok := latestMedia[vin]
	if !ok {
		return nil
	}
	copied := *m
	return &copied
}

func parseECPrivateKey(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not ECDSA")
		}
		return ecKey, nil
	}

	return nil, fmt.Errorf("failed to parse private key as SEC1 or PKCS#8")
}

func getFloat(data map[string]interface{}, key string) (float64, bool) {
	v, ok := data[key]
	if !ok { return 0, false }
	switch val := v.(type) {
	case float64: return val, true
	case int: return float64(val), true
	case json.Number:
		if f, err := val.Float64(); err == nil { return f, true }
	}
	return 0, false
}

func getInt(data map[string]interface{}, key string) (int, bool) {
	v, ok := data[key]
	if !ok { return 0, false }
	switch val := v.(type) {
	case float64: return int(val), true
	case int: return val, true
	case json.Number:
		if i, err := val.Int64(); err == nil { return int(i), true }
	}
	return 0, false
}

func getString(data map[string]interface{}, key string) (string, bool) {
	v, ok := data[key]
	if !ok { return "", false }
	s, ok := v.(string)
	return s, ok
}

func getBool(data map[string]interface{}, key string) (bool, bool) {
	v, ok := data[key]
	if !ok { return false, false }
	b, ok := v.(bool)
	return b, ok
}

// milesToKm converts miles to kilometers, rounded to 1 decimal place
func milesToKm(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return math.Round(v*1.60934*10) / 10
}

// Single-return value helpers for nested JSON format {"key": {"value": actual_value}}
func getFloatVal(data map[string]interface{}, key string) float64 {
	v, _ := getFloat(data, key)
	return v
}

func getIntVal(data map[string]interface{}, key string) int {
	v, _ := getInt(data, key)
	return v
}

func getStringVal(data map[string]interface{}, key string) string {
	v, _ := getString(data, key)
	return v
}

func getBoolVal(data map[string]interface{}, key string) bool {
	v, _ := getBool(data, key)
	return v
}
