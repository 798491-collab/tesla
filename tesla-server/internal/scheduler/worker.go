package scheduler

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"tesla-server/internal/charging"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/redis"
	vstate "tesla-server/internal/state"
	"tesla-server/internal/trip"
	"tesla-server/internal/vehicle"
	"tesla-server/internal/ws"
	"tesla-server/models"
	"time"
)

type pollState string

const (
	pollSleeping  pollState = "sleeping"
	pollWaking    pollState = "waking"
	pollOnline    pollState = "online"
	pollDriving   pollState = "driving"
	pollCharging  pollState = "charging"
	pollClimateOn pollState = "climate_on"
	pollUpdating  pollState = "updating"
	pollOffline   pollState = "offline"
)

type pollConfig struct {
	Interval       time.Duration
	UseLightweight bool
}

var pollConfigs = map[pollState]pollConfig{
	pollSleeping:  {Interval: 90 * time.Second, UseLightweight: true},
	pollWaking:    {Interval: 5 * time.Second, UseLightweight: false},
	pollOnline:    {Interval: 25 * time.Second, UseLightweight: false},
	pollDriving:   {Interval: 2 * time.Second, UseLightweight: false},
	pollCharging:  {Interval: 10 * time.Second, UseLightweight: false},
	pollClimateOn: {Interval: 5 * time.Second, UseLightweight: false},
	pollUpdating:  {Interval: 30 * time.Second, UseLightweight: false},
	pollOffline:   {Interval: 90 * time.Second, UseLightweight: true},
}

const (
	maxConsecutiveFailures = 3
	maxWakingAttempts      = 5
	wakingSuccessRequired  = 2
	sleepHintThreshold     = 3
	idleTimeout            = 10 * time.Minute
	onlineLockDuration     = 30 * time.Second
	minVehicleDataInterval = 3 * time.Second
)

type VehicleWorker struct {
	VIN                   string
	state                 pollState
	failCount             int
	wakingAttempts        int
	wakingSuccessCount    int
	sleepHintCount        int
	lastActiveTime        time.Time
	onlineLockUntil       time.Time
	lastVehicleDataCallAt time.Time
	cancelFunc            context.CancelFunc
	pollNow               chan struct{}
}

func NewVehicleWorker(vin string, cancel context.CancelFunc) *VehicleWorker {
	w := &VehicleWorker{
		VIN:                vin,
		state:              pollSleeping,
		failCount:          0,
		wakingAttempts:     0,
		wakingSuccessCount: 0,
		sleepHintCount:     0,
		lastActiveTime:     time.Now(),
		onlineLockUntil:    time.Time{},
		cancelFunc:         cancel,
		pollNow:            make(chan struct{}, 1),
	}

	var lastState fleet.SimpleVehicleData
	if err := redis.GetVehicleState(vin, &lastState); err == nil {
		if lastState.Online && lastState.State != "asleep" && lastState.State != "offline" {
			w.state = pollWaking
		}
	}

	return w
}

func (w *VehicleWorker) Cancel() {
	w.cancelFunc()
}

func (w *VehicleWorker) SignalPoll() {
	select {
	case w.pollNow <- struct{}{}:
	default:
	}
}

func (w *VehicleWorker) SignalActivity() {
	w.lastActiveTime = time.Now()
	w.sleepHintCount = 0
	w.failCount = 0
	if w.state == pollSleeping || w.state == pollOffline {
		w.transitionTo(pollWaking)
		w.wakingAttempts = 0
		w.wakingSuccessCount = 0
		w.SignalPoll()
	}
}

func (w *VehicleWorker) State() pollState {
	return w.state
}

func (w *VehicleWorker) isInOnlineLock() bool {
	return !w.onlineLockUntil.IsZero() && time.Now().Before(w.onlineLockUntil)
}

func (w *VehicleWorker) canDowngrade() bool {
	return !w.isInOnlineLock()
}

func (w *VehicleWorker) tryTransitionToSleeping(reason string) {
	if !w.canDowngrade() {
		log.Printf("[Worker] %s: would downgrade to sleeping (%s) but in online lock, deferring", w.VIN, reason)
		return
	}
	log.Printf("[Worker] %s: downgrading to sleeping (%s)", w.VIN, reason)
	w.transitionTo(pollSleeping)

	redis.SetVehicleOnline(w.VIN, false)
	database.DB.Model(&models.TeslaVehicle{}).
		Where("vin = ?", w.VIN).
		Update("online_state", "asleep")
}

func (w *VehicleWorker) Run(ctx context.Context) {
	cfg := pollConfigs[w.state]
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Printf("[Worker] %s started in state %s (interval: %v)", w.VIN, w.state, cfg.Interval)

	w.pollOnce()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Worker] %s stopped", w.VIN)
			return
		case <-w.pollNow:
			w.pollOnce()
			newCfg := pollConfigs[w.state]
			ticker.Reset(newCfg.Interval)
			if newCfg.Interval != cfg.Interval {
				cfg = newCfg
				log.Printf("[Worker] %s interval changed to %v (state: %s)", w.VIN, cfg.Interval, w.state)
			}
		case <-ticker.C:
			w.pollOnce()
			newCfg := pollConfigs[w.state]
			if newCfg.Interval != cfg.Interval {
				cfg = newCfg
				ticker.Reset(cfg.Interval)
				log.Printf("[Worker] %s interval changed to %v (state: %s)", w.VIN, cfg.Interval, w.state)
			}
		}
	}
}

func (w *VehicleWorker) pollOnce() {
	cfg := pollConfigs[w.state]
	if cfg.UseLightweight {
		w.pollLightweight()
	} else {
		if !w.checkVehicleDataCooldown() {
			return
		}
		w.pollFullData()
	}
}

func (w *VehicleWorker) checkVehicleDataCooldown() bool {
	if time.Since(w.lastVehicleDataCallAt) < minVehicleDataInterval {
		return false
	}
	return true
}

func (w *VehicleWorker) pollLightweight() {
	mapping, err := vehicle.GetVehicleMapping(w.VIN)
	if err != nil {
		log.Printf("[Worker] %s: mapping failed: %v", w.VIN, err)
		return
	}

	lightData, err := fleet.GetVehicleStateLightweight(mapping.AccessToken, mapping.VehicleTag)
	if err != nil {
		log.Printf("[Worker] %s: lightweight probe failed: %v", w.VIN, err)
		return
	}

	currentState := lightData.State
	if currentState == "" {
		if lightData.Online {
			currentState = "online"
		} else {
			currentState = "offline"
		}
	}

	if currentState == "asleep" || currentState == "offline" {
		w.sleepHintCount++
		log.Printf("[Worker] %s: /vehicles hint=%s (sleep hints: %d/%d)", w.VIN, currentState, w.sleepHintCount, sleepHintThreshold)

		vstate.UpdateFromLightweight(w.VIN, currentState, false, "lightweight_poll")

		redis.UpdateVehicleStateFields(w.VIN, map[string]interface{}{
			"state":  currentState,
			"online": false,
		})
		redis.SetVehicleOnline(w.VIN, false)
		ws.BroadcastOnlineState(w.VIN, currentState, false)

		if w.sleepHintCount >= sleepHintThreshold {
			w.tryTransitionToSleeping("sleep hints confirmed")
		}
		return
	}

	w.sleepHintCount = 0
	vstate.UpdateFromLightweight(w.VIN, currentState, true, "lightweight_poll")
	redis.UpdateVehicleStateFields(w.VIN, map[string]interface{}{
		"state":  currentState,
		"online": true,
	})
	redis.SetVehicleOnline(w.VIN, true)
	database.DB.Model(&models.TeslaVehicle{}).
		Where("vin = ?", w.VIN).
		Update("online_state", currentState)

	ws.BroadcastOnlineState(w.VIN, currentState, true)

	log.Printf("[Worker] %s: /vehicles reports online, entering waking confirmation", w.VIN)
	w.wakingAttempts = 0
	w.wakingSuccessCount = 0
	w.transitionTo(pollWaking)
}

func (w *VehicleWorker) pollFullData() {
	if w.state == pollWaking {
		w.pollWakingConfirm()
		return
	}

	w.checkPendingDowngrade()
	w.checkIdleTimeout()

	mapping, err := vehicle.GetVehicleMapping(w.VIN)
	if err != nil {
		log.Printf("[Worker] %s: mapping failed: %v", w.VIN, err)
		w.onFailure()
		return
	}

	w.lastVehicleDataCallAt = time.Now()
	data, err := fleet.GetVehicleState(mapping.AccessToken, mapping.VehicleTag)
	if err != nil {
		w.handleFullDataError(err)
		return
	}

	w.failCount = 0
	w.sleepHintCount = 0
	data.VIN = w.VIN

	preserveLastLocation(w.VIN, data)

	data.StateOutput = vstate.UpdateFromFullData(w.VIN, &vstate.VehicleDataInput{
		Speed:         data.Speed,
		Gear:          data.Gear,
		ChargingState: data.ChargingState,
		Supercharging: data.Supercharging,
		Soc:           data.Soc,
		ChargePower:   data.ChargePower,
		MinutesToFull: data.MinutesToFull,
		Locked:        data.Locked,
		DoorOpen:      data.DoorOpen,
	}, "api_poll")

	redis.SetVehicleState(w.VIN, data)
	redis.SetVehicleOnline(w.VIN, data.Online)
	redis.SetVehicleCharging(w.VIN, data.ChargingState == "Charging")

	ws.BroadcastVehicleState(w.VIN, data)

	w.updateActivityFromData(data)

	trip.ProcessDrivingState(w.VIN, data)
	charging.ProcessChargingState(w.VIN, data)

	saveVehicleState(w.VIN, data)

	newState := w.derivePollStateSafe(data)
	w.transitionTo(newState)

	database.DB.Model(&models.TeslaVehicle{}).
		Where("vin = ?", w.VIN).
		Update("online_state", data.State)
}

func (w *VehicleWorker) pollWakingConfirm() {
	w.wakingAttempts++
	log.Printf("[Worker] %s: waking confirm attempt %d/%d (success: %d/%d)", w.VIN, w.wakingAttempts, maxWakingAttempts, w.wakingSuccessCount, wakingSuccessRequired)

	mapping, err := vehicle.GetVehicleMapping(w.VIN)
	if err != nil {
		log.Printf("[Worker] %s: mapping failed during waking: %v", w.VIN, err)
		w.onWakingFailed()
		return
	}

	w.lastVehicleDataCallAt = time.Now()
	data, err := fleet.GetVehicleState(mapping.AccessToken, mapping.VehicleTag)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "vehicle asleep") ||
			strings.Contains(errMsg, "vehicle unavailable") ||
			strings.Contains(errMsg, "vehicle is offline") ||
			strings.Contains(errMsg, "vehicle timeout") {
			log.Printf("[Worker] %s: waking confirm - vehicle not really online (attempt %d)", w.VIN, w.wakingAttempts)
			w.wakingSuccessCount = 0
			if w.wakingAttempts >= maxWakingAttempts {
				w.onWakingFailed()
			}
			return
		}

		log.Printf("[Worker] %s: waking confirm error (transient): %v", w.VIN, err)
		if w.wakingAttempts >= maxWakingAttempts {
			w.onWakingFailed()
		}
		return
	}

	w.wakingSuccessCount++
	log.Printf("[Worker] %s: waking confirm success %d/%d", w.VIN, w.wakingSuccessCount, wakingSuccessRequired)

	data.VIN = w.VIN
	preserveLastLocation(w.VIN, data)
	data.StateOutput = vstate.UpdateFromFullData(w.VIN, &vstate.VehicleDataInput{
		Speed:         data.Speed,
		Gear:          data.Gear,
		ChargingState: data.ChargingState,
		Supercharging: data.Supercharging,
		Soc:           data.Soc,
		ChargePower:   data.ChargePower,
		MinutesToFull: data.MinutesToFull,
		Locked:        data.Locked,
		DoorOpen:      data.DoorOpen,
	}, "waking_confirm")
	redis.SetVehicleState(w.VIN, data)
	redis.SetVehicleOnline(w.VIN, data.Online)
	redis.SetVehicleCharging(w.VIN, data.ChargingState == "Charging")
	ws.BroadcastVehicleState(w.VIN, data)
	trip.ProcessDrivingState(w.VIN, data)
	charging.ProcessChargingState(w.VIN, data)
	saveVehicleState(w.VIN, data)

	database.DB.Model(&models.TeslaVehicle{}).
		Where("vin = ?", w.VIN).
		Update("online_state", data.State)

	if w.wakingSuccessCount >= wakingSuccessRequired {
		log.Printf("[Worker] %s: waking STABLE confirmed (%d consecutive successes), entering online with %v lock", w.VIN, w.wakingSuccessCount, onlineLockDuration)
		w.failCount = 0
		w.wakingAttempts = 0
		w.wakingSuccessCount = 0
		w.sleepHintCount = 0
		w.lastActiveTime = time.Now()
		w.onlineLockUntil = time.Now().Add(onlineLockDuration)
		w.transitionTo(pollOnline)
	}
}

func (w *VehicleWorker) onWakingFailed() {
	log.Printf("[Worker] %s: waking failed after %d attempts, returning to sleeping mode", w.VIN, w.wakingAttempts)
	w.wakingAttempts = 0
	w.wakingSuccessCount = 0
	w.transitionTo(pollSleeping)

	redis.SetVehicleOnline(w.VIN, false)
	database.DB.Model(&models.TeslaVehicle{}).
		Where("vin = ?", w.VIN).
		Update("online_state", "asleep")
}

func (w *VehicleWorker) handleFullDataError(err error) {
	errMsg := err.Error()

	if strings.Contains(errMsg, "vehicle asleep") ||
		strings.Contains(errMsg, "vehicle unavailable") ||
		strings.Contains(errMsg, "vehicle is offline") ||
		strings.Contains(errMsg, "vehicle timeout") {

		w.sleepHintCount++
		log.Printf("[Worker] %s: vehicle_data reports offline (sleep hints: %d/%d, inLock=%v)", w.VIN, w.sleepHintCount, sleepHintThreshold, w.isInOnlineLock())

		if w.sleepHintCount >= sleepHintThreshold {
			w.tryTransitionToSleeping("vehicle_data offline confirmed")
			w.failCount = 0
		} else {
			log.Printf("[Worker] %s: offline hint not confirmed yet - staying in current state", w.VIN)
		}
		return
	}

	log.Printf("[Worker] %s: vehicle_data transient error: %v", w.VIN, err)
	w.failCount++
	log.Printf("[Worker] %s: transient failure %d/%d (inLock=%v)", w.VIN, w.failCount, maxConsecutiveFailures, w.isInOnlineLock())

	if w.failCount >= maxConsecutiveFailures {
		w.tryTransitionToSleeping("consecutive transient failures")
		w.failCount = 0
	}
}

func (w *VehicleWorker) onFailure() {
	w.failCount++
	log.Printf("[Worker] %s: failure %d/%d (inLock=%v)", w.VIN, w.failCount, maxConsecutiveFailures, w.isInOnlineLock())

	if w.failCount >= maxConsecutiveFailures {
		w.tryTransitionToSleeping("consecutive failures")
		w.failCount = 0
	}
}

func (w *VehicleWorker) checkPendingDowngrade() {
	if w.isInOnlineLock() {
		return
	}

	if w.sleepHintCount >= sleepHintThreshold {
		log.Printf("[Worker] %s: online lock expired with %d pending sleep hints, downgrading", w.VIN, w.sleepHintCount)
		w.tryTransitionToSleeping("pending sleep hints after lock expired")
		w.failCount = 0
		return
	}

	if w.failCount >= maxConsecutiveFailures {
		log.Printf("[Worker] %s: online lock expired with %d pending failures, downgrading", w.VIN, w.failCount)
		w.tryTransitionToSleeping("pending failures after lock expired")
		w.failCount = 0
	}
}

func (w *VehicleWorker) updateActivityFromData(data *fleet.SimpleVehicleData) {
	if data.Gear == "D" || data.Gear == "R" || data.Gear == "N" {
		w.lastActiveTime = time.Now()
		return
	}
	if data.Speed > 0 {
		w.lastActiveTime = time.Now()
		return
	}
	if data.ChargingState == "Charging" {
		w.lastActiveTime = time.Now()
		return
	}
	if data.IsClimateOn {
		w.lastActiveTime = time.Now()
		return
	}
}

func (w *VehicleWorker) checkIdleTimeout() {
	if w.state == pollDriving || w.state == pollCharging || w.state == pollClimateOn {
		return
	}

	if w.state != pollOnline && w.state != pollUpdating {
		return
	}

	if w.isInOnlineLock() {
		return
	}

	var lastState fleet.SimpleVehicleData
	if err := redis.GetVehicleState(w.VIN, &lastState); err == nil {
		if lastState.ChargingState == "Charging" {
			return
		}
		if lastState.IsClimateOn {
			return
		}
		if lastState.Online && lastState.Speed == 0 && lastState.Gear == "P" {
			return
		}
	}

	idle := time.Since(w.lastActiveTime)
	if idle >= idleTimeout {
		log.Printf("[Worker] %s: idle for %v (threshold: %v), returning to sleeping mode", w.VIN, idle, idleTimeout)
		w.transitionTo(pollSleeping)

		redis.SetVehicleOnline(w.VIN, false)
		database.DB.Model(&models.TeslaVehicle{}).
			Where("vin = ?", w.VIN).
			Update("online_state", "asleep")
	}
}

func (w *VehicleWorker) derivePollStateSafe(data *fleet.SimpleVehicleData) pollState {
	if w.isInOnlineLock() {
		if data.Gear == "D" || data.Gear == "R" || data.Gear == "N" {
			w.onlineLockUntil = time.Time{}
			return pollDriving
		}
		if data.ChargingState == "Charging" {
			w.onlineLockUntil = time.Time{}
			return pollCharging
		}
		return pollOnline
	}

	w.onlineLockUntil = time.Time{}

	return derivePollState(data)
}

func (w *VehicleWorker) transitionTo(newState pollState) {
	if w.state == newState {
		return
	}

	oldState := w.state
	w.state = newState
	log.Printf("[Worker] %s: %s -> %s (interval: %v)", w.VIN, oldState, newState, pollConfigs[newState].Interval)

	ws.BroadcastPollState(w.VIN, string(newState))

	if newState == pollSleeping {
		w.handleTransitionToSleeping()
		w.onlineLockUntil = time.Time{}
		w.sleepHintCount = 0
		w.failCount = 0
	}

	if newState == pollWaking {
		w.wakingAttempts = 0
		w.wakingSuccessCount = 0
		w.onlineLockUntil = time.Time{}
	}

	if newState == pollOnline || newState == pollDriving || newState == pollCharging || newState == pollClimateOn {
		w.sleepHintCount = 0
		w.failCount = 0
	}
}

func (w *VehicleWorker) handleTransitionToSleeping() {
	var lastState fleet.SimpleVehicleData
	if err := redis.GetVehicleState(w.VIN, &lastState); err != nil {
		return
	}

	changed := false

	if lastState.Gear == "D" || lastState.Gear == "R" || lastState.Gear == "N" {
		lastState.Gear = "P"
		lastState.Speed = 0
		lastState.State = "asleep"
		changed = true
		trip.ProcessDrivingState(w.VIN, &lastState)
	}

	if lastState.ChargingState == "Charging" {
		lastState.ChargingState = "Complete"
		lastState.State = "asleep"
		changed = true
		charging.ProcessChargingState(w.VIN, &lastState)
	}

	if changed {
		redis.SetVehicleState(w.VIN, &lastState)
	}
}

func derivePollState(data *fleet.SimpleVehicleData) pollState {
	if data.State == "asleep" || data.State == "offline" {
		return pollSleeping
	}

	if data.Gear == "D" || data.Gear == "R" || data.Gear == "N" {
		return pollDriving
	}

	if data.ChargingState == "Charging" {
		return pollCharging
	}

	if data.IsClimateOn && (data.Gear == "P" || data.Gear == "") {
		return pollClimateOn
	}

	if data.State == "updating" {
		return pollUpdating
	}

	return pollOnline
}

var lastStateJSON = make(map[string]string)
var lastStateMu sync.Mutex

func saveVehicleState(vin string, data *fleet.SimpleVehicleData) {
	lastStateMu.Lock()
	lastJSON, exists := lastStateJSON[vin]
	lastStateMu.Unlock()

	currentState := models.VehicleTelemetry{
		VIN:           vin,
		BatteryLevel:  data.Soc,
		ChargingState: data.ChargingState,
		Speed:         data.Speed,
		ShiftState:    data.Gear,
		Latitude:      data.Latitude,
		Longitude:     data.Longitude,
		Odometer:      data.OdometerKm,
		Locked:        data.Locked,
		InsideTemp:    data.InsideTemp,
		OutsideTemp:   data.OutsideTemp,
		IsACOn:        data.IsACOn,
		Online:        data.Online,
		RecordedAt:    time.Now(),
	}

	currentJSON, _ := json.Marshal(currentState)

	if exists && string(currentJSON) == lastJSON {
		return
	}

	hasSignificantChange := !exists
	if exists {
		var last models.VehicleTelemetry
		json.Unmarshal([]byte(lastJSON), &last)
		hasSignificantChange = last.ShiftState != currentState.ShiftState ||
			last.ChargingState != currentState.ChargingState ||
			last.Online != currentState.Online ||
			last.Locked != currentState.Locked ||
			last.IsACOn != currentState.IsACOn ||
			last.BatteryLevel != currentState.BatteryLevel
	}

	if hasSignificantChange {
		database.DB.Create(&currentState)
	}

	lastStateMu.Lock()
	lastStateJSON[vin] = string(currentJSON)
	lastStateMu.Unlock()
}

func preserveLastLocation(vin string, data *fleet.SimpleVehicleData) {
	if data.Latitude != 0 && data.Longitude != 0 {
		return
	}

	var lastState fleet.SimpleVehicleData
	if err := redis.GetVehicleState(vin, &lastState); err != nil {
		return
	}

	if lastState.Latitude != 0 && lastState.Longitude != 0 {
		data.Latitude = lastState.Latitude
		data.Longitude = lastState.Longitude
		if data.Heading == 0 && lastState.Heading != 0 {
			data.Heading = lastState.Heading
		}
	}
}
