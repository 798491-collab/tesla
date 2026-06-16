package scheduler

import (
	"context"
	"log"
	"sync"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/redis"
	vstate "tesla-server/internal/state"
	"tesla-server/internal/tesla"
	"tesla-server/internal/vehicle"
	"tesla-server/models"
	"time"

	"github.com/robfig/cron/v3"
)

type vehicleManager struct {
	workers map[string]*VehicleWorker
	mu      sync.RWMutex
	cron    *cron.Cron
}

var mgr *vehicleManager

func InitDiscovery() {
	mgr = &vehicleManager{
		workers: make(map[string]*VehicleWorker),
		cron:    cron.New(cron.WithSeconds()),
	}

	mgr.cron.AddFunc("0 */5 * * * *", func() {
		checkAndRefreshTokens()
	})

	mgr.cron.AddFunc("0 0 3 * * *", func() {
		cleanupOldStates()
	})

	mgr.cron.Start()

	startAllWorkers()
}

func startAllWorkers() {
	var vehicles []models.TeslaVehicle
	database.DB.Where("bind_status = 1").Find(&vehicles)

	for _, v := range vehicles {
		vehicle.RefreshVehicleMapping(v.VIN)
		mgr.startWorker(v.VIN)
	}
}

func (m *vehicleManager) startWorker(vin string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workers[vin]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := NewVehicleWorker(vin, cancel)
	m.workers[vin] = w

	go w.Run(ctx)
	log.Printf("[Manager] Started worker for %s", vin)
}

func (m *vehicleManager) stopWorker(vin string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if w, exists := m.workers[vin]; exists {
		w.Cancel()
		delete(m.workers, vin)
		log.Printf("[Manager] Stopped worker for %s", vin)
	}
}

func (m *vehicleManager) signalWorker(vin string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if w, exists := m.workers[vin]; exists {
		w.SignalPoll()
	}
}

func (m *vehicleManager) signalActivity(vin string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if w, exists := m.workers[vin]; exists {
		w.SignalActivity()
	}
}

func StartVehicleDiscovery(vin string) {
	if mgr == nil {
		return
	}

	mgr.mu.Lock()
	if w, exists := mgr.workers[vin]; exists {
		mgr.mu.Unlock()
		w.SignalPoll()
		return
	}
	mgr.mu.Unlock()

	mgr.startWorker(vin)
}

func GetWorkerPollState(vin string) string {
	if mgr == nil {
		return "unknown"
	}
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	if w, exists := mgr.workers[vin]; exists {
		return string(w.State())
	}
	return "stopped"
}

func StopVehicleDiscovery(vin string) {
	if mgr != nil {
		mgr.stopWorker(vin)
	}
}

func GetVehicleStateFromRedis(vin string) (*fleet.SimpleVehicleData, error) {
	var state fleet.SimpleVehicleData
	if err := redis.GetVehicleState(vin, &state); err != nil {
		return nil, err
	}
	if state.VIN == "" {
		state.VIN = vin
	}
	if state.StateOutput == nil {
		state.StateOutput = vstate.GetOutput(vin, &vstate.VehicleDataInput{
			Speed:         state.Speed,
			Gear:          state.Gear,
			ChargingState: state.ChargingState,
			Supercharging: state.Supercharging,
			Soc:           state.Soc,
			ChargePower:   state.ChargePower,
			MinutesToFull: state.MinutesToFull,
			Locked:        state.Locked,
			DoorOpen:      state.DoorOpen,
			CruiseState:        state.CruiseState,
			AutosteerState:     state.AutosteerState,
			CruiseControlState: state.CruiseControlState,
		})
	}
	return &state, nil
}

func RefreshVehicleState(vin string) (*fleet.SimpleVehicleData, error) {
	mapping, err := vehicle.GetVehicleMapping(vin)
	if err != nil {
		return nil, err
	}

	data, err := fleet.GetVehicleState(mapping.AccessToken, mapping.VehicleTag)
	if err != nil {
		return nil, err
	}

	data.VIN = vin
	data.ID = 1
	data.UpdatedAt = time.Now()

	if data.Latitude == 0 && data.Longitude == 0 {
		var lastState fleet.SimpleVehicleData
		if err := redis.GetVehicleState(vin, &lastState); err == nil {
			if lastState.Latitude != 0 && lastState.Longitude != 0 {
				data.Latitude = lastState.Latitude
				data.Longitude = lastState.Longitude
				if data.Heading == 0 && lastState.Heading != 0 {
					data.Heading = lastState.Heading
				}
			}
		}
	}

	redis.SetVehicleState(vin, data)
	redis.SetVehicleOnline(vin, data.Online)
	redis.SetVehicleCharging(vin, data.ChargingState == "Charging")

	data.StateOutput = vstate.UpdateFromFullData(vin, &vstate.VehicleDataInput{
		Speed:         data.Speed,
		Gear:          data.Gear,
		ChargingState: data.ChargingState,
		Supercharging: data.Supercharging,
		Soc:           data.Soc,
		ChargePower:   data.ChargePower,
		MinutesToFull: data.MinutesToFull,
		Locked:        data.Locked,
		DoorOpen:      data.DoorOpen,
		CruiseState:        data.CruiseState,
		AutosteerState:     data.AutosteerState,
		CruiseControlState: data.CruiseControlState,
	}, "manual_refresh")

	return data, nil
}

func WakeVehicle(vin string) error {
	mapping, err := vehicle.GetVehicleMapping(vin)
	if err != nil {
		return err
	}

	err = fleet.WakeUp(mapping.AccessToken, mapping.VehicleTag)
	if err != nil {
		return err
	}

	if mgr != nil {
		mgr.signalActivity(vin)
	}

	return nil
}

func SignalVehicleActivity(vin string) {
	if mgr != nil {
		mgr.signalActivity(vin)
	}
}

func checkAndRefreshTokens() {
	var vehicles []models.TeslaVehicle
	database.DB.Where("bind_status = 1").Find(&vehicles)

	now := time.Now().Unix()
	threshold := now + 1800

	for _, v := range vehicles {
		var oauthAccount models.TeslaOAuthAccount
		if err := database.DB.Where("user_id = ? AND tesla_uid = ?", v.UserID, v.TeslaUID).First(&oauthAccount).Error; err != nil {
			log.Printf("[TokenRefresh] OAuth account not found for %s (user_id=%d, tesla_uid=%s): %v", v.VIN, v.UserID, v.TeslaUID, err)
			continue
		}

		if oauthAccount.TokenInvalid {
			log.Printf("[TokenRefresh] %s token marked invalid, skipping (user must re-authorize)", v.VIN)
			continue
		}

		if oauthAccount.ExpiresAt < threshold {
			log.Printf("[TokenRefresh] %s token expiring soon (expires_at=%d, threshold=%d), refreshing...", v.VIN, oauthAccount.ExpiresAt, threshold)
			_, err := tesla.RefreshTokenForVehicle(v.VIN)
			if err != nil {
				log.Printf("[TokenRefresh] Failed for %s: %v", v.VIN, err)
				continue
			}
			log.Printf("[TokenRefresh] Token refreshed for %s", v.VIN)
		}
	}
}

func cleanupOldStates() {
	cutoff := time.Now().AddDate(0, 0, -30)
	database.DB.Where("recorded_at < ?", cutoff).Delete(&models.VehicleTelemetry{})
}
