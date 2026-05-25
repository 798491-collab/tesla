package polling

import (
	"tesla-server/internal/fleet"
	"tesla-server/internal/redis"
	"tesla-server/internal/scheduler"
	"tesla-server/internal/vehicle"
)

func Init() {
	scheduler.InitDiscovery()
}

func StartVehiclePolling(vin string) {
	scheduler.StartVehicleDiscovery(vin)
}

func StopVehiclePolling(vin string) {
	scheduler.StopVehicleDiscovery(vin)
}

func GetVehicleState(vin string) (*fleet.SimpleVehicleData, error) {
	return scheduler.GetVehicleStateFromRedis(vin)
}

func RefreshVehicleState(vin string) (*fleet.SimpleVehicleData, error) {
	data, err := scheduler.RefreshVehicleState(vin)
	if err != nil {
		return nil, err
	}

	vehicle.RefreshVehicleMapping(vin)
	scheduler.StartVehicleDiscovery(vin)

	return data, nil
}

func WakeVehicle(vin string) error {
	return scheduler.WakeVehicle(vin)
}

func SetPollingState(vin string, state interface{}) error {
	return redis.SetPollingState(vin, state)
}

func GetPollingState(vin string, dest interface{}) error {
	return redis.GetPollingState(vin, dest)
}

func DeletePollingState(vin string) error {
	return redis.DeletePollingState(vin)
}

func HasWakeLock(vin string) (bool, error) {
	return redis.HasWakeLock(vin)
}

func GetWorkerPollState(vin string) string {
	return scheduler.GetWorkerPollState(vin)
}

func SignalActivity(vin string) {
	scheduler.SignalVehicleActivity(vin)
}

func FormatVehicleState(state *fleet.SimpleVehicleData) map[string]interface{} {
	if state == nil {
		return nil
	}
	return map[string]interface{}{
		"vin":            state.VIN,
		"online":         state.Online,
		"state":          state.State,
		"driving":        state.Driving,
		"charging":       state.Charging,
		"soc":            state.Soc,
		"usable_soc":     state.UsableSoc,
		"range_km":       state.RangeKm,
		"speed":          state.Speed,
		"gear":           state.Gear,
		"power":          state.Power,
		"charging_state": state.ChargingState,
		"charge_speed":   state.ChargeSpeed,
		"charge_power":   state.ChargePower,
		"ampere":         state.Ampere,
		"voltage":        state.Voltage,
		"added_energy":   state.AddedEnergy,
		"minutes_to_full": state.MinutesToFull,
		"supercharging":  state.Supercharging,
		"inside_temp":    state.InsideTemp,
		"outside_temp":   state.OutsideTemp,
		"locked":         state.Locked,
		"is_ac_on":       state.IsACOn,
		"is_climate_on":  state.IsClimateOn,
		"sentry_mode":    state.SentryMode,
		"windows_open":   state.WindowsOpen,
		"door_open":      state.DoorOpen,
		"latitude":       state.Latitude,
		"longitude":      state.Longitude,
		"heading":        state.Heading,
		"odometer_km":    state.OdometerKm,
		"version":        state.Version,
	}
}

func CheckWakeLock(vin string) error {
	return nil
}
