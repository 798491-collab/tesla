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
	"strconv"
	"strings"
	"sync"
	"tesla-server/internal/geo"
	"tesla-server/internal/redis"
	"tesla-server/internal/state"
	"tesla-server/internal/ws"

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
	latestMedia = make(map[string]map[string]interface{})
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
			log.Printf("[Telemetry] Private key parse failed: %v", err)
		} else {
			privateKey = key
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/1/vehicles/", handleTelemetry)
	mux.HandleFunc("/", handleTelemetryRoot)

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
		} else {
			defaultCA = defaultProdCA
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
			log.Printf("[Telemetry] mTLS server listening on %s (ca=%s)", addr, caMode)
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("[Telemetry] Server error: %v", err)
			}
		}()
	} else {
		go func() {
			log.Printf("[Telemetry] HTTP server listening on %s (no TLS)", addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Telemetry] Server error: %v", err)
			}
		}()
	}

	return nil
}

// handleTelemetryRoot 处理车辆直接连接根路径的遥测请求
// Tesla 车辆通过 mTLS WebSocket 连接 wss://hostname:8443/，路径为根路径
// VIN 从 mTLS 客户端证书中提取
func handleTelemetryRoot(w http.ResponseWriter, r *http.Request) {
	// 从 mTLS 客户端证书提取 VIN
	var vin string
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		clientType, deviceID, err := messages.CreateIdentityFromCert(r.TLS.PeerCertificates[0])
		if err != nil {
			log.Printf("[Telemetry] [mTLS] Cert identity extract failed: %v", err)
			http.Error(w, "invalid client certificate", http.StatusForbidden)
			return
		}
		vin = deviceID
		log.Printf("[Telemetry] [mTLS] Cert verified: VIN=%s, type=%s, path=%s", vin, clientType, r.URL.Path)
	} else {
		log.Printf("[Telemetry] No client cert, path=%s", r.URL.Path)
		http.Error(w, "client certificate required", http.StatusForbidden)
		return
	}

	// WebSocket 升级
	if isWebSocketRequest(r) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Telemetry] [WS] Upgrade failed for %s: %v", vin, err)
			return
		}
		handleTelemetryWS(vin, conn)
		return
	}

	// HTTP POST
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Telemetry] [HTTP] Read body failed for %s: %v", vin, err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	log.Printf("[Telemetry] [HTTP] Received POST for %s, body_len=%d", vin, len(body))
	processRawPayload(vin, body)
	w.WriteHeader(http.StatusOK)
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		log.Printf("[Telemetry] Invalid path: %s", r.URL.Path)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	vin := pathParts[3]

	// 节点1: mTLS 客户端证书验证
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		clientType, deviceID, err := messages.CreateIdentityFromCert(r.TLS.PeerCertificates[0])
		if err != nil {
			log.Printf("[Telemetry] [mTLS] Cert identity extract failed for %s: %v", vin, err)
			http.Error(w, "invalid client certificate", http.StatusForbidden)
			return
		}
		if deviceID != vin {
			log.Printf("[Telemetry] [mTLS] VIN mismatch: URL=%s, Cert=%s (type=%s)", vin, deviceID, clientType)
			http.Error(w, "VIN mismatch", http.StatusForbidden)
			return
		}
		log.Printf("[Telemetry] [mTLS] Cert verified: VIN=%s, type=%s", vin, clientType)
	} else if r.TLS != nil {
		log.Printf("[Telemetry] [mTLS] No client cert for VIN=%s (TLS connection without peer cert)", vin)
	}

	// 节点2: WebSocket 升级判断
	if isWebSocketRequest(r) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Telemetry] [WS] Upgrade failed for %s: %v", vin, err)
			return
		}
		handleTelemetryWS(vin, conn)
		return
	}

	// 节点3: HTTP POST 请求处理
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Telemetry] [HTTP] Read body failed for %s: %v", vin, err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	log.Printf("[Telemetry] [HTTP] Received POST for %s, body_len=%d", vin, len(body))
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

	// 节点4: WebSocket 连接建立
	log.Printf("[Telemetry] [WS] Connected: VIN=%s, remote=%s", vin, conn.RemoteAddr())

	redis.SetVehicleStatus(vin, &redis.VehicleStatus{
		Online: true,
		Source: "telemetry",
	})

	defer func() {
		conn.Close()
		activeConnsMu.Lock()
		delete(activeConns, vin)
		activeConnsMu.Unlock()
		log.Printf("[Telemetry] [WS] Disconnected: VIN=%s", vin)

		// 通知前端遥测断开，车辆可能已睡眠
		redis.SetVehicleStatus(vin, &redis.VehicleStatus{
			Online: false,
			Source: "telemetry",
		})
		redis.UpdateVehicleStateFields(vin, map[string]interface{}{
			"online": false,
		})
		ws.DefaultHub.BroadcastToVIN(vin, "online_state", map[string]interface{}{
			"state":  "asleep",
			"online": false,
		})
	}()

	msgCount := 0
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[Telemetry] [WS] Unexpected close for %s: %v", vin, err)
			} else {
				log.Printf("[Telemetry] [WS] Disconnected: VIN=%s (total_msgs=%d)", vin, msgCount)
			}
			break
		}

		msgCount++

		// 节点5: WebSocket 消息接收
		if messageType == websocket.BinaryMessage {
			go processRawPayload(vin, payload)
		} else {
			log.Printf("[Telemetry] [WS] Non-binary message for %s: type=%d, len=%d", vin, messageType, len(payload))
		}
	}
}

func processRawPayload(vin string, body []byte) {
	// 节点6: 原始数据解码
	// Tesla 官方 fleet-telemetry 使用 Flatbuffers 封装格式
	// 通过 messages.StreamMessageFromBytes 解析信封，提取 MessageTopic 和 Payload
	if len(body) == 0 {
		return
	}

	streamMsg, err := messages.StreamMessageFromBytes(body)
	if err != nil {
		log.Printf("[Telemetry] [Decode] StreamMessage parse failed for %s: %v (body_len=%d), hex=% x", vin, err, len(body), body[:min(len(body), 20)])
		return
	}

	topic := streamMsg.Topic()
	payloadBytes := streamMsg.Payload
	txid := string(streamMsg.TXID)

	// 存储原始二进制数据，用于事后分析
	RecordRaw(vin, topic, txid, body)

	log.Printf("[Telemetry] [Decode] StreamMessage OK for %s: topic=%s, txid=%s, payload_len=%d", vin, topic, txid, len(payloadBytes))

	switch topic {
	case "V": // 车辆遥测数据
		var payload protos.Payload
		if err := proto.Unmarshal(payloadBytes, &payload); err != nil {
			log.Printf("[Telemetry] [Decode] Payload unmarshal failed for %s: %v (len=%d)", vin, err, len(payloadBytes))
			return
		}
		log.Printf("[Telemetry] [Decode] Payload OK for %s: %d data points", vin, len(payload.Data))
		processProtobufTelemetry(vin, &payload)
	case "alerts":
		var alerts protos.VehicleAlerts
		if err := proto.Unmarshal(payloadBytes, &alerts); err != nil {
			log.Printf("[Telemetry] [Decode] Alerts unmarshal failed for %s: %v", vin, err)
			return
		}
		log.Printf("[Telemetry] [Decode] Alerts OK for %s: %d alerts", vin, len(alerts.Alerts))
	case "errors":
		var vehErrors protos.VehicleErrors
		if err := proto.Unmarshal(payloadBytes, &vehErrors); err != nil {
			log.Printf("[Telemetry] [Decode] Errors unmarshal failed for %s: %v", vin, err)
			return
		}
		log.Printf("[Telemetry] [Decode] Errors OK for %s: %d errors", vin, len(vehErrors.Errors))
	case "connectivity":
		var conn protos.VehicleConnectivity
		if err := proto.Unmarshal(payloadBytes, &conn); err != nil {
			log.Printf("[Telemetry] [Decode] Connectivity unmarshal failed for %s: %v", vin, err)
			return
		}
		log.Printf("[Telemetry] [Decode] Connectivity OK for %s: status=%v", vin, conn.Status)
	default:
		log.Printf("[Telemetry] [Decode] Unknown topic %q for %s, payload_len=%d", topic, vin, len(payloadBytes))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleJSONTelemetry(vin string, body []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("[Telemetry] [Decode] JSON parse also failed for %s: %v", vin, err)
		return
	}
	log.Printf("[Telemetry] [Decode] JSON OK for %s: %d keys", vin, len(data))

	realtimeFields := map[string]interface{}{}
	stateFields := map[string]interface{}{}
	mediaFields := map[string]interface{}{}

	hasRealtime := false
	hasState := false
	hasMedia := false

	if v, ok := getFloat(data, "VehicleSpeed"); ok {
		realtimeFields["speed"] = v * 1.60934 // mph → km/h
		hasRealtime = true
	}
	if v, ok := getString(data, "Gear"); ok {
		realtimeFields["gear"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "Power"); ok {
		realtimeFields["power"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "PedalPosition"); ok {
		realtimeFields["pedal_position"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "CruiseSetSpeed"); ok {
		realtimeFields["cruise_set_speed"] = v * 1.60934 // mph → km/h
		hasRealtime = true
	}
	if v, ok := getFloat(data, "LateralAcceleration"); ok {
		realtimeFields["lateral_acceleration"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "LongitudinalAcceleration"); ok {
		realtimeFields["longitudinal_acceleration"] = v
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
		realtimeFields["latitude"] = lat
		realtimeFields["longitude"] = lng
	}
	if v, ok := getInt(data, "GpsHeading"); ok {
		realtimeFields["heading"] = v
		hasRealtime = true
	}
	if v, ok := getInt(data, "GpsState"); ok {
		realtimeFields["gps_state"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "Soc"); ok {
		realtimeFields["soc"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "BatteryLevel"); ok {
		realtimeFields["battery_level"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "DCChargingPower"); ok {
		realtimeFields["dc_charging_power"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "ACChargingPower"); ok {
		realtimeFields["ac_charging_power"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "PackVoltage"); ok {
		realtimeFields["pack_voltage"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "PackCurrent"); ok {
		realtimeFields["pack_current"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "EnergyRemaining"); ok {
		realtimeFields["energy_remaining"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "ChargeAmps"); ok {
		realtimeFields["charge_amps"] = v
		hasRealtime = true
	}
	if v, ok := getFloat(data, "ChargerVoltage"); ok {
		realtimeFields["charger_voltage"] = v
		hasRealtime = true
	}
	if v, ok := getString(data, "ChargeState"); ok {
		realtimeFields["charge_state"] = v
		hasRealtime = true
	}
	if v, ok := getString(data, "DetailedChargeState"); ok {
		realtimeFields["charge_state"] = v
		hasRealtime = true
	}
	if v, ok := getBool(data, "FastChargerPresent"); ok {
		realtimeFields["fast_charger_present"] = v
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
		// door_open 只计算四个车门，不含 trunk/frunk（与 REST 通道一致）
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
	if v, ok := getString(data, "ChargePortLatch"); ok {
		stateFields["charge_port_latch"] = v
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
		case "RdWindow":
			stateFields["rd_window"] = getBoolVal(v, "value")
			hasState = true
		case "RpWindow":
			stateFields["rp_window"] = getBoolVal(v, "value")
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
				realtimeFields["gps_state"] = val
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
				realtimeFields["charge_state"] = s
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
		mediaFields["media_playback_status"] = v
		hasMedia = true
	} else if v, ok := getFloat(data, "MediaPlaybackStatus"); ok {
		// JSON 降级路径中，MediaPlaybackStatus 可能是数字枚举值
		switch int(v) {
		case 1:
			mediaFields["media_playback_status"] = "Stopped"
		case 2:
			mediaFields["media_playback_status"] = "Playing"
		case 3:
			mediaFields["media_playback_status"] = "Paused"
		default:
			mediaFields["media_playback_status"] = "Unknown"
		}
		hasMedia = true
	}
	if v, ok := getString(data, "MediaPlaybackSource"); ok {
		mediaFields["media_audio_source"] = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaAudioSource"); ok {
		mediaFields["media_audio_source"] = v
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaAudioVolume"); ok {
		mediaFields["media_volume"] = int(v)
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaVolume"); ok {
		mediaFields["media_volume"] = int(v)
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaAudioVolumeIncrement"); ok {
		mediaFields["media_audio_volume_increment"] = int(v)
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaAudioVolumeMax"); ok {
		mediaFields["media_audio_volume_max"] = int(v)
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingTitle"); ok {
		mediaFields["now_playing_title"] = v
		hasMedia = true
	}
	if v, ok := getString(data, "NowPlayingTitle"); ok {
		mediaFields["now_playing_title"] = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingArtist"); ok {
		mediaFields["now_playing_artist"] = v
		hasMedia = true
	}
	if v, ok := getString(data, "NowPlayingArtist"); ok {
		mediaFields["now_playing_artist"] = v
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingAlbum"); ok {
		mediaFields["now_playing_album"] = v
		hasMedia = true
	}
	if v, ok := getString(data, "NowPlayingAlbum"); ok {
		mediaFields["now_playing_album"] = v
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaNowPlayingDuration"); ok {
		mediaFields["now_playing_duration"] = int(v)
		hasMedia = true
	}
	if v, ok := getFloat(data, "MediaNowPlayingElapsed"); ok {
		mediaFields["now_playing_elapsed"] = int(v)
		hasMedia = true
	}
	if v, ok := getString(data, "MediaNowPlayingStation"); ok {
		mediaFields["now_playing_station"] = v
		hasMedia = true
	}

	// 节点7: JSON 数据分发
	if hasRealtime {
		log.Printf("[Telemetry] [Dispatch] JSON realtime for %s: %d fields", vin, len(realtimeFields))
		updateRealtimeFields(vin, realtimeFields)
	}
	if hasState {
		log.Printf("[Telemetry] [Dispatch] JSON state for %s: %d fields", vin, len(stateFields))
		updateVehicleStateFields(vin, stateFields)
	}
	if hasMedia {
		log.Printf("[Telemetry] [Dispatch] JSON media for %s: %d fields", vin, len(mediaFields))
		updateMediaFields(vin, mediaFields)
	}
}

func processProtobufTelemetry(vin string, payload *protos.Payload) {

	realtimeFields := map[string]interface{}{}
	stateFields := map[string]interface{}{}
	mediaFields := map[string]interface{}{}

	hasRealtime := false
	hasState := false
	hasMedia := false

	for _, datum := range payload.Data {
		if datum.Value == nil || datum.Value.GetInvalid() {
			continue
		}

		switch datum.Key {
		case protos.Field_VehicleSpeed:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["speed"] = v * 1.60934 // mph → km/h
				log.Printf("[Telemetry] [Decode] VehicleSpeed: type=%T, raw=%.4f, kmh=%.2f", datum.Value.GetValue(), v, realtimeFields["speed"])
				hasRealtime = true
			}

		case protos.Field_Gear:
			switch datum.Value.GetShiftStateValue() {
			case protos.ShiftState_ShiftStateP:
				realtimeFields["gear"] = "P"
				hasRealtime = true
			case protos.ShiftState_ShiftStateR:
				realtimeFields["gear"] = "R"
				hasRealtime = true
			case protos.ShiftState_ShiftStateN:
				realtimeFields["gear"] = "N"
				hasRealtime = true
			case protos.ShiftState_ShiftStateD:
				realtimeFields["gear"] = "D"
				hasRealtime = true
			default:
				// 未知挡位状态，跳过不写入
			}

		case protos.Field_PedalPosition:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["pedal_position"] = v
				hasRealtime = true
			}
		case protos.Field_CruiseSetSpeed:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["cruise_set_speed"] = v * 1.60934 // mph → km/h
				hasRealtime = true
			}
		case protos.Field_LateralAcceleration:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["lateral_acceleration"] = v
				hasRealtime = true
			}
		case protos.Field_LongitudinalAcceleration:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["longitudinal_acceleration"] = v
				hasRealtime = true
			}
		case protos.Field_Location:
			loc := datum.Value.GetLocationValue()
			if loc != nil {
				// Convert WGS-84 to GCJ-02 for China maps
				lat, lng := geo.WGS84ToGCJ02(loc.GetLatitude(), loc.GetLongitude())
				realtimeFields["latitude"] = lat
				realtimeFields["longitude"] = lng
				hasRealtime = true

			}
		case protos.Field_GpsHeading:
			if v, ok := getIntValue(datum.Value); ok {
				realtimeFields["heading"] = v
				hasRealtime = true
			}
		case protos.Field_GpsState:
			if v, ok := getIntValue(datum.Value); ok {
				realtimeFields["gps_state"] = v
				hasRealtime = true
			}
		case protos.Field_Soc:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["soc"] = v
				hasRealtime = true
			}

		case protos.Field_BatteryLevel:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["battery_level"] = v
				hasRealtime = true
			}

		case protos.Field_DCChargingPower:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["dc_charging_power"] = v
				hasRealtime = true
			}
		case protos.Field_ACChargingPower:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["ac_charging_power"] = v
				hasRealtime = true
			}
		case protos.Field_PackVoltage:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["pack_voltage"] = v
				hasRealtime = true
			}
		case protos.Field_PackCurrent:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["pack_current"] = v
				hasRealtime = true
			}
		case protos.Field_EnergyRemaining:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["energy_remaining"] = v
				hasRealtime = true
			}
		case protos.Field_ChargeAmps:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["charge_amps"] = v
				hasRealtime = true
			}
		case protos.Field_ChargerVoltage:
			if v, ok := getFloat64Value(datum.Value); ok {
				realtimeFields["charger_voltage"] = v
				hasRealtime = true
			}
		case protos.Field_ChargeState:
			realtimeFields["charge_state"] = chargingStateToString(datum.Value.GetChargingValue())
			hasRealtime = true
		case protos.Field_DetailedChargeState:
			realtimeFields["charge_state"] = detailedChargeStateToString(datum.Value.GetDetailedChargeStateValue())
			hasRealtime = true
		case protos.Field_FastChargerPresent:
			realtimeFields["fast_charger_present"] = datum.Value.GetBooleanValue()
			hasRealtime = true

		case protos.Field_Locked:
			stateFields["locked"] = datum.Value.GetBooleanValue()
			hasState = true

		case protos.Field_DoorState:
			doors := datum.Value.GetDoorValue()
			if doors != nil {
				doorOpen := false
				stateFields["door_fl"] = doors.GetDriverFront()
				if doors.GetDriverFront() {
					doorOpen = true
				}
				stateFields["door_rl"] = doors.GetDriverRear()
				if doors.GetDriverRear() {
					doorOpen = true
				}
				stateFields["door_fr"] = doors.GetPassengerFront()
				if doors.GetPassengerFront() {
					doorOpen = true
				}
				stateFields["door_rr"] = doors.GetPassengerRear()
				if doors.GetPassengerRear() {
					doorOpen = true
				}
				stateFields["trunk_open"] = doors.GetTrunkRear()
				stateFields["frunk_open"] = doors.GetTrunkFront()
				stateFields["door_open"] = doorOpen
				hasState = true

			}
		case protos.Field_SentryMode:
			sentryState := datum.Value.GetSentryModeStateValue()
			stateFields["sentry_mode"] = sentryState != protos.SentryModeState_SentryModeStateOff && sentryState != protos.SentryModeState_SentryModeStateUnknown
			hasState = true

		case protos.Field_InsideTemp:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["inside_temp"] = v
				hasState = true
			}
		case protos.Field_OutsideTemp:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["outside_temp"] = v
				hasState = true
			}
		case protos.Field_ChargePortDoorOpen:
			stateFields["charge_port_door_open"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ChargePortLatch:
			latchVal := datum.Value.GetChargePortLatchValue()
			stateFields["charge_port_latch"] = chargePortLatchToString(latchVal)
			hasState = true
		case protos.Field_ChargeLimitSoc:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["charge_limit_soc"] = v
				hasState = true
			}
		case protos.Field_TpmsPressureFl:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_fl"] = v
				hasState = true
			}

		case protos.Field_TpmsPressureFr:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_fr"] = v
				hasState = true
			}

		case protos.Field_TpmsPressureRl:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_rl"] = v
				hasState = true
			}

		case protos.Field_TpmsPressureRr:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_rr"] = v
				hasState = true
			}


		case protos.Field_Odometer:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["odometer_km"] = v * 1.60934
				hasState = true
			}

		case protos.Field_CenterDisplay:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["center_display_state"] = v
				hasState = true
			}
		case protos.Field_HvacPower:
			hvacState := datum.Value.GetHvacPowerValue()
			hvacOn := hvacState != protos.HvacPowerState_HvacPowerStateOff && hvacState != protos.HvacPowerState_HvacPowerStateUnknown
			stateFields["hvac_power"] = hvacOn
			stateFields["is_ac_on"] = hvacOn
			stateFields["is_climate_on"] = hvacOn
			hasState = true
		case protos.Field_HvacLeftTemperatureRequest:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["driver_temp_setting"] = v
				hasState = true
			}
		case protos.Field_HvacRightTemperatureRequest:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["passenger_temp_setting"] = v
				hasState = true
			}
		case protos.Field_DefrostMode:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["defrost_mode"] = v
				hasState = true
			}
		case protos.Field_ChargeRateMilePerHour:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["charge_speed"] = v * 1.60934
				hasState = true
			}
		case protos.Field_IdealBatteryRange:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["range_km"] = v * 1.60934
				hasState = true
			}
		case protos.Field_EstBatteryRange:
			if v, ok := getFloat64Value(datum.Value); ok {
				r := v * 1.60934
				if stateFields["range_km"] == nil {
					stateFields["range_km"] = r
				}
				hasState = true
			}
		case protos.Field_BrakePedal:
			stateFields["brake_pedal"] = datum.Value.GetBooleanValue()
			hasState = true

		case protos.Field_DriveRail:
			stateFields["drive_rail"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_SeatHeaterLeft:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["seat_heater_left"] = v
				hasState = true
			}
		case protos.Field_SeatHeaterRight:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["seat_heater_right"] = v
				hasState = true
			}
		case protos.Field_SeatHeaterRearLeft:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["seat_heater_rear_left"] = v
				hasState = true
			}
		case protos.Field_SeatHeaterRearRight:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["seat_heater_rear_right"] = v
				hasState = true
			}
		case protos.Field_SeatHeaterRearCenter:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["seat_heater_rear_center"] = v
				hasState = true
			}
		case protos.Field_DriverSeatBelt:
			stateFields["driver_seat_belt"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DriverSeatOccupied:
			stateFields["driver_seat_occupied"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_FdWindow:
			stateFields["fd_window"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_FpWindow:
			stateFields["fp_window"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_RdWindow:
			stateFields["rd_window"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_RpWindow:
			stateFields["rp_window"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HvacACEnabled:
			stateFields["hvac_ac_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HvacSteeringWheelHeatLevel:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["steering_wheel_heater"] = v > 0
				stateFields["hvac_steering_wheel_heat_level"] = v
				hasState = true
			}
		case protos.Field_HvacSteeringWheelHeatAuto:
			stateFields["hvac_steering_wheel_heat_auto"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_HvacFanSpeed:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["hvac_fan_speed"] = v
				hasState = true
			}
		case protos.Field_HvacAutoMode:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["hvac_auto_mode"] = v
				hasState = true
			}
		case protos.Field_ClimateKeeperMode:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["climate_keeper_mode"] = v
				hasState = true
			}
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
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["climate_seat_cooling_front_left"] = v
				hasState = true
			}
		case protos.Field_ClimateSeatCoolingFrontRight:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["climate_seat_cooling_front_right"] = v
				hasState = true
			}
		case protos.Field_CabinOverheatProtectionMode:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["cabin_overheat_protection_mode"] = v
				hasState = true
			}
		case protos.Field_CabinOverheatProtectionTemperatureLimit:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["cabin_overheat_protection_temperature_limit"] = v
				hasState = true
			}
		case protos.Field_BatteryHeaterOn:
			stateFields["battery_heater_on"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DCChargingEnergyIn:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["dc_charging_energy_in"] = v
				hasState = true
			}
		case protos.Field_ACChargingEnergyIn:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["ac_charging_energy_in"] = v
				hasState = true
			}
		case protos.Field_EstimatedHoursToChargeTermination:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["minutes_to_full"] = v * 60
				hasState = true
			}
		case protos.Field_FastChargerType:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["fast_charger_type"] = v
				hasState = true
			}
		case protos.Field_ChargingCableType:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["charging_cable_type"] = v
				hasState = true
			}
		case protos.Field_ChargeEnableRequest:
			stateFields["charge_enable_request"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ChargeCurrentRequest:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["charge_current_request"] = v
				hasState = true
			}
		case protos.Field_ChargeCurrentRequestMax:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["charge_current_request_max"] = v
				hasState = true
			}
		case protos.Field_ChargerPhases:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["charger_phases"] = v
				hasState = true
			}
		case protos.Field_ChargePortColdWeatherMode:
			stateFields["charge_port_cold_weather_mode"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_DestinationLocation:
			loc := datum.Value.GetLocationValue()
			if loc != nil {
				stateFields["destination_latitude"] = loc.GetLatitude()
				stateFields["destination_longitude"] = loc.GetLongitude()
				hasState = true
			}
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
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["homelink_device_count"] = v
				hasState = true
			}
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
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["bms_state"] = v
				hasState = true
			}
		case protos.Field_BmsFullchargecomplete:
			stateFields["bms_full_charge_complete"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LifetimeEnergyUsed:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["lifetime_energy_used"] = v
				hasState = true
			}
		case protos.Field_CurrentLimitMph:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["current_limit_mph"] = v
				hasState = true
			}
		case protos.Field_CruiseFollowDistance:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["cruise_follow_distance"] = v
				hasState = true
			}
		case protos.Field_AutomaticBlindSpotCamera:
			stateFields["automatic_blind_spot_camera"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_BlindSpotCollisionWarningChime:
			stateFields["blind_spot_collision_warning_chime"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ForwardCollisionWarning:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["forward_collision_warning"] = v
				hasState = true
			}
		case protos.Field_LaneDepartureAvoidance:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["lane_departure_avoidance"] = v
				hasState = true
			}
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
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["brick_voltage_max"] = v
				hasState = true
			}
		case protos.Field_BrickVoltageMin:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["brick_voltage_min"] = v
				hasState = true
			}
		case protos.Field_IsolationResistance:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["isolation_resistance"] = v
				hasState = true
			}
		case protos.Field_Hvil:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["hvil"] = v
				hasState = true
			}

		case protos.Field_MediaPlaybackStatus:
			mediaFields["media_playback_status"] = mediaStatusToString(datum.Value.GetMediaStatusValue())
			hasMedia = true
		case protos.Field_MediaPlaybackSource:
			mediaFields["media_audio_source"] = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaAudioVolume:
			if v, ok := getIntValue(datum.Value); ok {
				mediaFields["media_volume"] = v
				hasMedia = true
			}
		case protos.Field_MediaAudioVolumeIncrement:
			if v, ok := getIntValue(datum.Value); ok {
				mediaFields["media_audio_volume_increment"] = v
				hasMedia = true
			}
		case protos.Field_MediaAudioVolumeMax:
			if v, ok := getIntValue(datum.Value); ok {
				mediaFields["media_audio_volume_max"] = v
				hasMedia = true
			}
		case protos.Field_MediaNowPlayingDuration:
			if v, ok := getIntValue(datum.Value); ok {
				mediaFields["now_playing_duration"] = v
				hasMedia = true
			}
		case protos.Field_MediaNowPlayingElapsed:
			if v, ok := getIntValue(datum.Value); ok {
				mediaFields["now_playing_elapsed"] = v
				hasMedia = true
			}
		case protos.Field_MediaNowPlayingArtist:
			mediaFields["now_playing_artist"] = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaNowPlayingTitle:
			mediaFields["now_playing_title"] = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaNowPlayingAlbum:
			mediaFields["now_playing_album"] = datum.Value.GetStringValue()
			hasMedia = true
		case protos.Field_MediaNowPlayingStation:
			mediaFields["now_playing_station"] = datum.Value.GetStringValue()
			hasMedia = true

		// ========== 充电调度/预处理 ==========
		case protos.Field_TimeToFullCharge:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["time_to_full_charge"] = v
				hasState = true
			}
		case protos.Field_ScheduledChargingStartTime:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["scheduled_charging_start_time"] = v
				hasState = true
			}
		case protos.Field_ScheduledChargingPending:
			stateFields["scheduled_charging_pending"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ScheduledDepartureTime:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["scheduled_departure_time"] = v
				hasState = true
			}
		case protos.Field_PreconditioningEnabled:
			stateFields["preconditioning_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_ScheduledChargingMode:
			mode := datum.Value.GetScheduledChargingModeValue()
			switch mode {
			case protos.ScheduledChargingModeValue_ScheduledChargingModeOff:
				stateFields["scheduled_charging_mode"] = "Off"
			case protos.ScheduledChargingModeValue_ScheduledChargingModeStartAt:
				stateFields["scheduled_charging_mode"] = "StartAt"
			case protos.ScheduledChargingModeValue_ScheduledChargingModeDepartBy:
				stateFields["scheduled_charging_mode"] = "DepartBy"
			default:
				stateFields["scheduled_charging_mode"] = "Unknown"
			}
			hasState = true
		case protos.Field_ChargePort:
			cp := datum.Value.GetChargePortValue()
			switch cp {
			case protos.ChargePortValue_ChargePortUS:
				stateFields["charge_port"] = "US"
			case protos.ChargePortValue_ChargePortEU:
				stateFields["charge_port"] = "EU"
			case protos.ChargePortValue_ChargePortGB:
				stateFields["charge_port"] = "GB"
			case protos.ChargePortValue_ChargePortCCS:
				stateFields["charge_port"] = "CCS"
			default:
				stateFields["charge_port"] = "Unknown"
			}
			hasState = true
		case protos.Field_NotEnoughPowerToHeat:
			stateFields["not_enough_power_to_heat"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_SuperchargerSessionTripPlanner:
			stateFields["supercharger_session_trip_planner"] = datum.Value.GetBooleanValue()
			hasState = true

		// ========== 安全/驾驶辅助 ==========
		case protos.Field_SpeedLimitMode:
			stateFields["speed_limit_mode"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_PassengerSeatBelt:
			buckle := datum.Value.GetBuckleStatusValue()
			stateFields["passenger_seat_belt"] = buckle == protos.BuckleStatus_BuckleStatusLatched
			hasState = true
		case protos.Field_BrakePedalPos:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["brake_pedal_pos"] = v
				hasState = true
			}
		case protos.Field_PinToDriveEnabled:
			stateFields["pin_to_drive_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_PairedPhoneKeyAndKeyFobQty:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["paired_phone_key_and_key_fob_qty"] = v
				hasState = true
			}
		case protos.Field_SpeedLimitWarning:
			level := datum.Value.GetSpeedAssistLevelValue()
			switch level {
			case protos.SpeedAssistLevel_SpeedAssistLevelNone:
				stateFields["speed_limit_warning"] = "None"
			case protos.SpeedAssistLevel_SpeedAssistLevelDisplay:
				stateFields["speed_limit_warning"] = "Display"
			case protos.SpeedAssistLevel_SpeedAssistLevelChime:
				stateFields["speed_limit_warning"] = "Chime"
			default:
				stateFields["speed_limit_warning"] = "Unknown"
			}
			hasState = true
		case protos.Field_ValetModeEnabled:
			stateFields["valet_mode_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LifetimeEnergyGainedRegen:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["lifetime_energy_gained_regen"] = v
				hasState = true
			}
		case protos.Field_RearDefrostEnabled:
			stateFields["rear_defrost_enabled"] = datum.Value.GetBooleanValue()
			hasState = true

		// ========== 导航/行程 ==========
		case protos.Field_RouteLastUpdated:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["route_last_updated"] = v
				hasState = true
			}
		case protos.Field_MilesToArrival:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["km_to_arrival"] = v * 1.60934
				hasState = true
			}
		case protos.Field_MinutesToArrival:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["minutes_to_arrival"] = v
				hasState = true
			}
		case protos.Field_OriginLocation:
			loc := datum.Value.GetLocationValue()
			if loc != nil {
				stateFields["origin_latitude"] = loc.GetLatitude()
				stateFields["origin_longitude"] = loc.GetLongitude()
				hasState = true
			}
		case protos.Field_ExpectedEnergyPercentAtTripArrival:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["expected_energy_percent_at_arrival"] = v
				hasState = true
			}
		case protos.Field_RouteTrafficMinutesDelay:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["route_traffic_minutes_delay"] = v
				hasState = true
			}

		// ========== 里程（英里→公里） ==========
		case protos.Field_MilesSinceReset:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["km_since_reset"] = v * 1.60934
				hasState = true
			}
		case protos.Field_SelfDrivingMilesSinceReset:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["self_driving_km_since_reset"] = v * 1.60934
				hasState = true
			}
		case protos.Field_RatedRange:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["rated_range_km"] = v * 1.60934
				hasState = true
			}

		// ========== 车辆信息 ==========
		case protos.Field_VehicleName:
			stateFields["vehicle_name"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_Trim:
			stateFields["trim"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_RoofColor:
			stateFields["roof_color"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_WheelType:
			stateFields["wheel_type"] = datum.Value.GetStringValue()
			hasState = true
		case protos.Field_EuropeVehicle:
			stateFields["europe_vehicle"] = datum.Value.GetBooleanValue()
			hasState = true

		// ========== 地理围栏 ==========
		case protos.Field_LocatedAtHome:
			stateFields["located_at_home"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LocatedAtWork:
			stateFields["located_at_work"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_LocatedAtFavorite:
			stateFields["located_at_favorite"] = datum.Value.GetBooleanValue()
			hasState = true

		// ========== 电池诊断 ==========
		case protos.Field_NumBrickVoltageMax:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["num_brick_voltage_max"] = v
				hasState = true
			}
		case protos.Field_NumBrickVoltageMin:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["num_brick_voltage_min"] = v
				hasState = true
			}
		case protos.Field_NumModuleTempMax:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["num_module_temp_max"] = v
				hasState = true
			}
		case protos.Field_ModuleTempMax:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["module_temp_max"] = v
				hasState = true
			}
		case protos.Field_NumModuleTempMin:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["num_module_temp_min"] = v
				hasState = true
			}
		case protos.Field_ModuleTempMin:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["module_temp_min"] = v
				hasState = true
			}

		// ========== HVAC/气候 ==========
		case protos.Field_HvacFanStatus:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["hvac_fan_status"] = v
				hasState = true
			}
		case protos.Field_RearDisplayHvacEnabled:
			stateFields["rear_display_hvac_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_SeatVentEnabled:
			stateFields["seat_vent_enabled"] = datum.Value.GetBooleanValue()
			hasState = true

		// ========== Powershare (V2G/V2L) ==========
		case protos.Field_PowershareHoursLeft:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["powershare_hours_left"] = v
				hasState = true
			}
		case protos.Field_PowershareInstantaneousPowerKW:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["powershare_instantaneous_power_kw"] = v
				hasState = true
			}
		case protos.Field_PowershareStatus:
			ps := datum.Value.GetPowershareStateValue()
			stateFields["powershare_status"] = ps.String()
			hasState = true
		case protos.Field_PowershareStopReason:
			psr := datum.Value.GetPowershareStopReasonValue()
			stateFields["powershare_stop_reason"] = psr.String()
			hasState = true
		case protos.Field_PowershareType:
			pt := datum.Value.GetPowershareTypeValue()
			stateFields["powershare_type"] = pt.String()
			hasState = true

		// ========== 软件更新 ==========
		case protos.Field_SoftwareUpdateDownloadPercentComplete:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["software_update_download_percent"] = v
				hasState = true
			}
		case protos.Field_SoftwareUpdateExpectedDurationMinutes:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["software_update_expected_duration_minutes"] = v
				hasState = true
			}
		case protos.Field_SoftwareUpdateInstallationPercentComplete:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["software_update_installation_percent"] = v
				hasState = true
			}
		case protos.Field_SoftwareUpdateScheduledStartTime:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["software_update_scheduled_start_time"] = v
				hasState = true
			}
		case protos.Field_SoftwareUpdateVersion:
			stateFields["software_update_version"] = datum.Value.GetStringValue()
			hasState = true

		// ========== Cybertruck 专用 ==========
		case protos.Field_OffroadLightbarPresent:
			stateFields["offroad_lightbar_present"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_TonneauOpenPercent:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tonneau_open_percent"] = v
				hasState = true
			}
		case protos.Field_TonneauPosition:
			tp := datum.Value.GetTonneauPositionValue()
			stateFields["tonneau_position"] = tp.String()
			hasState = true
		case protos.Field_TonneauTentMode:
			ttm := datum.Value.GetTonneauTentModeValue()
			stateFields["tonneau_tent_mode"] = ttm.String()
			hasState = true

		// ========== TPMS 详细 ==========
		case protos.Field_TpmsLastSeenPressureTimeFl:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_last_seen_pressure_time_fl"] = v
				hasState = true
			}
		case protos.Field_TpmsLastSeenPressureTimeFr:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_last_seen_pressure_time_fr"] = v
				hasState = true
			}
		case protos.Field_TpmsLastSeenPressureTimeRl:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_last_seen_pressure_time_rl"] = v
				hasState = true
			}
		case protos.Field_TpmsLastSeenPressureTimeRr:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["tpms_last_seen_pressure_time_rr"] = v
				hasState = true
			}
		case protos.Field_TpmsHardWarnings:
			stateFields["tpms_hard_warnings"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_TpmsSoftWarnings:
			stateFields["tpms_soft_warnings"] = datum.Value.GetBooleanValue()
			hasState = true

		// ========== 其他 ==========
		case protos.Field_WiperHeatEnabled:
			stateFields["wiper_heat_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_RemoteStartEnabled:
			stateFields["remote_start_enabled"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_RearSeatHeaters:
			if v, ok := getIntValue(datum.Value); ok {
				stateFields["rear_seat_heaters"] = v
				hasState = true
			}
		case protos.Field_RightHandDrive:
			stateFields["right_hand_drive"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_SunroofInstalled:
			si := datum.Value.GetSunroofInstalledStateValue()
			stateFields["sunroof_installed"] = si.String()
			hasState = true
		case protos.Field_GuestModeMobileAccessState:
			gm := datum.Value.GetGuestModeMobileAccessValue()
			stateFields["guest_mode_mobile_access_state"] = gm.String()
			hasState = true

		// ========== 用户偏好设置 ==========
		case protos.Field_SettingDistanceUnit:
			du := datum.Value.GetDistanceUnitValue()
			stateFields["setting_distance_unit"] = du.String()
			hasState = true
		case protos.Field_SettingTemperatureUnit:
			tu := datum.Value.GetTemperatureUnitValue()
			stateFields["setting_temperature_unit"] = tu.String()
			hasState = true
		case protos.Field_Setting24HourTime:
			stateFields["setting_24_hour_time"] = datum.Value.GetBooleanValue()
			hasState = true
		case protos.Field_SettingTirePressureUnit:
			pu := datum.Value.GetPressureUnitValue()
			stateFields["setting_tire_pressure_unit"] = pu.String()
			hasState = true
		case protos.Field_SettingChargeUnit:
			cu := datum.Value.GetChargeUnitPreferenceValue()
			stateFields["setting_charge_unit"] = cu.String()
			hasState = true

		// ========== 驱动逆变器诊断（Di* 字段） ==========
		case protos.Field_DiStateR:
			stateFields["di_state_r"] = datum.Value.GetDriveInverterStateValue().String()
			hasState = true
		case protos.Field_DiStateF:
			stateFields["di_state_f"] = datum.Value.GetDriveInverterStateValue().String()
			hasState = true
		case protos.Field_DiStateREL:
			stateFields["di_state_rel"] = datum.Value.GetDriveInverterStateValue().String()
			hasState = true
		case protos.Field_DiStateRER:
			stateFields["di_state_rer"] = datum.Value.GetDriveInverterStateValue().String()
			hasState = true
		case protos.Field_DiHeatsinkTR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_heatsink_t_r"] = v
				hasState = true
			}
		case protos.Field_DiHeatsinkTF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_heatsink_t_f"] = v
				hasState = true
			}
		case protos.Field_DiHeatsinkTREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_heatsink_t_rel"] = v
				hasState = true
			}
		case protos.Field_DiHeatsinkTRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_heatsink_t_rer"] = v
				hasState = true
			}
		case protos.Field_DiAxleSpeedR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_axle_speed_r"] = v
				hasState = true
			}
		case protos.Field_DiAxleSpeedF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_axle_speed_f"] = v
				hasState = true
			}
		case protos.Field_DiAxleSpeedREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_axle_speed_rel"] = v
				hasState = true
			}
		case protos.Field_DiAxleSpeedRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_axle_speed_rer"] = v
				hasState = true
			}
		case protos.Field_DiTorquemotor:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_torque_motor"] = v
				hasState = true
			}
		case protos.Field_DiSlaveTorqueCmd:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_slave_torque_cmd"] = v
				hasState = true
			}
		case protos.Field_DiTorqueActualR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_torque_actual_r"] = v
				hasState = true
			}
		case protos.Field_DiTorqueActualF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_torque_actual_f"] = v
				hasState = true
			}
		case protos.Field_DiTorqueActualREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_torque_actual_rel"] = v
				hasState = true
			}
		case protos.Field_DiTorqueActualRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_torque_actual_rer"] = v
				hasState = true
			}
		case protos.Field_DiStatorTempR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_stator_temp_r"] = v
				hasState = true
			}
		case protos.Field_DiStatorTempF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_stator_temp_f"] = v
				hasState = true
			}
		case protos.Field_DiStatorTempREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_stator_temp_rel"] = v
				hasState = true
			}
		case protos.Field_DiStatorTempRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_stator_temp_rer"] = v
				hasState = true
			}
		case protos.Field_DiVBatR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_vbat_r"] = v
				hasState = true
			}
		case protos.Field_DiVBatF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_vbat_f"] = v
				hasState = true
			}
		case protos.Field_DiVBatREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_vbat_rel"] = v
				hasState = true
			}
		case protos.Field_DiVBatRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_vbat_rer"] = v
				hasState = true
			}
		case protos.Field_DiMotorCurrentR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_motor_current_r"] = v
				hasState = true
			}
		case protos.Field_DiMotorCurrentF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_motor_current_f"] = v
				hasState = true
			}
		case protos.Field_DiMotorCurrentREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_motor_current_rel"] = v
				hasState = true
			}
		case protos.Field_DiMotorCurrentRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_motor_current_rer"] = v
				hasState = true
			}
		case protos.Field_DiInverterTR:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_inverter_t_r"] = v
				hasState = true
			}
		case protos.Field_DiInverterTF:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_inverter_t_f"] = v
				hasState = true
			}
		case protos.Field_DiInverterTREL:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_inverter_t_rel"] = v
				hasState = true
			}
		case protos.Field_DiInverterTRER:
			if v, ok := getFloat64Value(datum.Value); ok {
				stateFields["di_inverter_t_rer"] = v
				hasState = true
			}

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
				// 未知类型跳过，不存入 fmt.Sprintf 的字符串
				continue
			}
			hasState = true

		}
	}

	// 节点8: Protobuf 数据分发（只推送实际收到的字段，不覆盖未推送的字段）
	if hasRealtime {
		log.Printf("[Telemetry] [Dispatch] Protobuf realtime for %s: %d fields", vin, len(realtimeFields))
		updateRealtimeFields(vin, realtimeFields)
	}
	if hasState {
		log.Printf("[Telemetry] [Dispatch] Protobuf state for %s: %d fields", vin, len(stateFields))
		updateVehicleStateFields(vin, stateFields)
	}
	if hasMedia {
		log.Printf("[Telemetry] [Dispatch] Protobuf media for %s: %d fields", vin, len(mediaFields))
		updateMediaFields(vin, mediaFields)
	}
}

func getFloat64Value(v *protos.Value) (float64, bool) {
	switch val := v.GetValue().(type) {
	case *protos.Value_DoubleValue:
		return val.DoubleValue, true
	case *protos.Value_FloatValue:
		return float64(val.FloatValue), true
	case *protos.Value_IntValue:
		return float64(val.IntValue), true
	case *protos.Value_LongValue:
		return float64(val.LongValue), true
	case *protos.Value_StringValue:
		// Tesla proto 注释: "Most Datums are strings and is the default format"
		// Field 179 之前的字段可能以 string_value 返回数值
		if f, err := strconv.ParseFloat(val.StringValue, 64); err == nil {
			return f, true
		}
		log.Printf("[Telemetry] [Decode] StringValue parse failed: %q", val.StringValue)
		return 0, false
	default:
		// 未知类型，跳过而非返回0
		log.Printf("[Telemetry] [Decode] getFloat64Value unhandled type: %T, value: %v", v.GetValue(), v.GetValue())
		return 0, false
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

func getIntValue(v *protos.Value) (int, bool) {
	switch val := v.GetValue().(type) {
	case *protos.Value_IntValue:
		return int(val.IntValue), true
	case *protos.Value_LongValue:
		return int(val.LongValue), true
	case *protos.Value_DoubleValue:
		return int(val.DoubleValue), true
	case *protos.Value_FloatValue:
		return int(val.FloatValue), true
	case *protos.Value_StringValue:
		// Tesla proto 注释: "Most Datums are strings and is the default format"
		if i, err := strconv.Atoi(val.StringValue); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(val.StringValue, 64); err == nil {
			return int(f), true
		}
		return 0, false
	default:
		return 0, false
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

func chargePortLatchToString(latch protos.ChargePortLatchValue) string {
	switch latch {
	case protos.ChargePortLatchValue_ChargePortLatchDisengaged:
		return "Disengaged"
	case protos.ChargePortLatchValue_ChargePortLatchEngaged:
		return "Engaged"
	case protos.ChargePortLatchValue_ChargePortLatchBlocking:
		return "Blocking"
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

func updateRealtimeFields(vin string, fields map[string]interface{}) {
	// 节点9: 实时数据增量写入（只更新实际推送的字段，不覆盖未推送的字段）
	RecordRealtime(vin, fields)

	if err := redis.UpdateVehicleRealtimeFields(vin, fields); err != nil {
		log.Printf("[Telemetry] [Store] Redis realtime error for %s: %v", vin, err)
	}

	redis.SetVehicleStatus(vin, &redis.VehicleStatus{
		Online: true,
		Source: "telemetry",
	})

	// 更新状态引擎（增量合并，保留上次已知值）
	stateOutput := state.UpdateFromTelemetry(vin, fields)
	if stateOutput != nil {
		fields["state_output"] = stateOutput
	}

	ws.BroadcastRealtimeUpdate(vin, fields)
}

func updateVehicleStateFields(vin string, fields map[string]interface{}) {
	// 节点10: 状态数据写入
	RecordState(vin, fields)

	if err := redis.UpdateVehicleStateFields(vin, fields); err != nil {
		log.Printf("[Telemetry] [Store] Redis state error for %s: %v", vin, err)
	}

	// 更新状态引擎（增量合并，保留上次已知值）
	stateOutput := state.UpdateFromTelemetry(vin, fields)
	if stateOutput != nil {
		fields["state_output"] = stateOutput
	}

	ws.BroadcastStateUpdate(vin, fields)
}

func updateMediaFields(vin string, fields map[string]interface{}) {
	// 节点11: 媒体数据增量写入（只更新实际推送的字段，不覆盖未推送的字段）
	RecordMedia(vin, fields)

	if err := redis.UpdateVehicleStateFields(vin, fields); err != nil {
		log.Printf("[Telemetry] [Store] Redis media error for %s: %v", vin, err)
	}

	// 更新状态引擎（增量合并，保留上次已知值）
	state.UpdateFromTelemetry(vin, fields)

	if ws.DefaultHub != nil {
		ws.DefaultHub.BroadcastToVIN(vin, "media_state", fields)
	}
}

func GetLatestMedia(vin string) map[string]interface{} {
	mediaMu.RLock()
	defer mediaMu.RUnlock()
	m, ok := latestMedia[vin]
	if !ok {
		return nil
	}
	copied := make(map[string]interface{}, len(m))
	for k, v := range m {
		copied[k] = v
	}
	return copied
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
