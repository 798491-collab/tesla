package fleet

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"tesla-server/config"
	"tesla-server/internal/geo"
	"tesla-server/internal/state"

	"github.com/go-resty/resty/v2"
)

// 单位转换常量
const mileToKm = 1.60934

// milesToKm 将英里转换为公里，保留1位小数
// 注意：不要依赖 gui_distance_units，Tesla API 的显示单位和底层单位经常不一致
func milesToKm(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return math.Round(v*mileToKm*10) / 10
}

// 生产级 HTTP Client 配置
// vehicle_data: 15~20s, wake_up: 30~45s, command: 20s
var (
	// 通用 client，用于 vehicle_data、command 等
	client = resty.New().
		SetTimeout(20 * time.Second).
		SetRetryCount(1).
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	// wake_up 专用 client，超时更长
	wakeUpClient = resty.New().
		SetTimeout(45 * time.Second).
		SetRetryCount(1).
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	// VCP 专用 client，跳过 TLS 证书验证（VCP 使用自签名证书）
	vcpClient = resty.New().
		SetTimeout(20 * time.Second).
		SetRetryCount(1).
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
)

// TeslaError Tesla API 错误响应
type TeslaError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// GUISettings GUI 设置
type GUISettings struct {
	GuiDistanceUnits    string `json:"gui_distance_units"`
	GuiTemperatureUnits string `json:"gui_temperature_units"`
	GuiChargeRateUnits  string `json:"gui_charge_rate_units"`
	GuiTimeFormat       string `json:"gui_time_format"`
}

// VehicleState 车辆状态（包含胎压，注意：胎压字段在 vehicle_state 下）
type VehicleState struct {
	Odometer           float64 `json:"odometer"`
	DF                 int     `json:"df"`          // 左前门 (0:关, 1:开)
	PF                 int     `json:"pf"`          // 右前门 (0:关, 1:开)
	DR                 int     `json:"dr"`          // 左后门 (0:关, 1:开)
	PR                 int     `json:"pr"`          // 右后门 (0:关, 1:开)
	FT                 int     `json:"ft"`          // 前备箱 (0:关, 1或2:开/解锁)
	RT                 int     `json:"rt"`          // 后备箱 (0:关, 1或2:开/解锁)
	FdWindow           int     `json:"fd_window"`   // 左前窗 (0:关, >0:开)
	FpWindow           int     `json:"fp_window"`   // 右前窗
	RdWindow           int     `json:"rd_window"`   // 左后窗
	RpWindow           int     `json:"rp_window"`   // 右后窗
	IsWindowClosed     bool    `json:"is_window_closed"` // 快捷字段：只要窗户没关严就是 false
	Locked             bool    `json:"locked"`
	SentryMode         bool    `json:"sentry_mode"`
	CarVersion         string  `json:"car_version"`
	CenterDisplayState int     `json:"center_display_state"`
	MirrorFolded       bool    `json:"mirror_folded"`
	LightsHazardsActive bool   `json:"lights_hazards_active"`
	GuestModeEnabled   bool    `json:"guest_mode_enabled"`
	ServiceMode        bool    `json:"service_mode"`
	TPMSPressureFL float64 `json:"tpms_pressure_fl"`
	TPMSPressureFR float64 `json:"tpms_pressure_fr"`
	TPMSPressureRL float64 `json:"tpms_pressure_rl"`
	TPMSPressureRR float64 `json:"tpms_pressure_rr"`
	SpeedLimitMode SpeedLimitMode `json:"speed_limit_mode"`
}

// ChargeState 充电状态
type ChargeState struct {
	BatteryLevel        int     `json:"battery_level"`
	UsableBatteryLevel  int     `json:"usable_battery_level"`
	BatteryRange        float64 `json:"battery_range"`
	ChargingState       string  `json:"charging_state"`
	ChargeRate          float64 `json:"charge_rate"`
	ChargePortOpen      bool    `json:"charge_port_door_open"`
	ChargePortLatch     string  `json:"charge_port_latch"`
	MinutesToFullCharge int     `json:"minutes_to_full_charge"`
	ChargeEnergyAdded   float64 `json:"charge_energy_added"`
	ChargeMilesAdded    float64 `json:"charge_miles_added_rated"`
	ChargerVoltage      int     `json:"charger_voltage"`
	ChargerCurrent      int     `json:"charger_current"`
	ChargerPower        int     `json:"charger_power"`
	FastChargerPresent  bool    `json:"fast_charger_present"`
	FastChargerType     string  `json:"fast_charger_type"`
	ChargeLimitSoc      int     `json:"charge_limit_soc"`
	BatteryHeaterOn     bool    `json:"battery_heater_on"`
	OutsideTemp         float64 `json:"outside_temp"`
}

// DriveState 驾驶状态
type DriveState struct {
	ShiftState        string  `json:"shift_state"`
	Speed             int     `json:"speed"`
	Power             int     `json:"power"`
	Heading           int     `json:"heading"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	BrakePedal        bool    `json:"brake_pedal"`
	DriveRail         bool    `json:"drive_rail"`
	AcceleratorPedalPos float64 `json:"accelerator_pedal_pos"`
	ActiveRouteLatitude       float64 `json:"active_route_latitude"`
	ActiveRouteLongitude      float64 `json:"active_route_longitude"`
	ActiveRouteName           string  `json:"active_route_name"`
	ActiveRouteMilesToArrival float64 `json:"active_route_miles_to_arrival"`
	ActiveRouteMinutesToArrival float64 `json:"active_route_minutes_to_arrival"`
	LightsHighBeams      bool    `json:"lights_high_beams"`
	LightsTurnSignal     string  `json:"lights_turn_signal"`
	CruiseState          string  `json:"cruise_state"`
	AutosteerState       string  `json:"autosteer_state"`
	CruiseControlState   string  `json:"cruise_control_state"`
	LaneKeepingState     string  `json:"lane_keeping_state"`
	ActiveRouteSpeedLimit float64 `json:"active_route_speed_limit"`
}

// SpeedLimitMode 限速模式
type SpeedLimitMode struct {
	Active         bool    `json:"active"`
	CurrentLimitMph float64 `json:"current_limit_mph"`
	MaxLimitMph    float64 `json:"max_limit_mph"`
	MinLimitMph    float64 `json:"min_limit_mph"`
	PinCodeSet     bool    `json:"pin_code_set"`
}

// FlexInt 可以同时解析 JSON 中的整数和字符串值
// Tesla API 某些字段（如 climate_keeper_mode, defrost_mode）可能返回 int 或 string
type FlexInt int

func (f *FlexInt) UnmarshalJSON(data []byte) error {
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*f = FlexInt(i)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "off", "":
			*f = 0
		case "on":
			*f = 1
		case "dog":
			*f = 2
		case "camp":
			*f = 3
		default:
			if v, err := strconv.Atoi(s); err == nil {
				*f = FlexInt(v)
			} else {
				*f = 0
			}
		}
		return nil
	}
	return fmt.Errorf("FlexInt: cannot unmarshal %s", string(data))
}

// ClimateState 气候状态
type ClimateState struct {
	InsideTemp              float64 `json:"inside_temp"`
	OutsideTemp             float64 `json:"outside_temp"`
	DriverTempSetting       float64 `json:"driver_temp_setting"`
	PassengerTempSetting    float64 `json:"passenger_temp_setting"`
	IsClimateOn             bool    `json:"is_climate_on"`
	IsAirConditioningOn     bool    `json:"is_air_conditioning_on"`
	AutoConditioningEnabled bool    `json:"auto_conditioning_enabled"`
	FanStatus               int     `json:"fan_status"`
	BatteryTemp             float64 `json:"battery_temp"`
	SeatHeaterLeft          int     `json:"seat_heater_left"`
	SeatHeaterRight         int     `json:"seat_heater_right"`
	SeatHeaterRearLeft      int     `json:"seat_heater_rear_left"`
	SeatHeaterRearRight     int     `json:"seat_heater_rear_right"`
	SeatHeaterRearCenter    int     `json:"seat_heater_rear_center"`
	SteeringWheelHeat       bool    `json:"steering_wheel_heat"`
	DefrostMode             FlexInt `json:"defrost_mode"`
	ClimateKeeperMode       FlexInt `json:"climate_keeper_mode"`
}

// VehicleConfig 车辆配置
type VehicleConfig struct {
	CarType       string `json:"car_type"`
	Trim          string `json:"trim"`
	ExteriorColor string `json:"exterior_color"`
	WheelType     string `json:"wheel_type"`
}

// VehicleDataResponse 完整车辆数据响应
type VehicleDataResponse struct {
	VehicleState VehicleState  `json:"vehicle_state"`
	ChargeState  ChargeState   `json:"charge_state"`
	DriveState   DriveState    `json:"drive_state"`
	ClimateState ClimateState  `json:"climate_state"`
	GUISettings  GUISettings   `json:"gui_settings"`
	MediaState   MediaState    `json:"media_state"`
	MediaInfo    MediaInfo     `json:"media_info"`
	VehicleConfig VehicleConfig `json:"vehicle_config"`
	// 注意：closures_state 在中国区很多账号不支持，已移除
}

// MediaState 媒体状态（REST API vehicle_data 的 media_state 端点）
// 注意：REST API 的 media_state 只返回 remote_control_enabled
// 丰富的媒体数据（now_playing 等）通过 Fleet Telemetry 的 media_info 类别推送
type MediaState struct {
	RemoteControlEnabled bool `json:"remote_control_enabled"`
}

// MediaInfo 媒体信息（REST API vehicle_data 的 media_info 端点）
// REST API 返回的字段名与遥测不同：now_playing_source vs MediaPlaybackSource
type MediaInfo struct {
	MediaPlaybackStatus string `json:"media_playback_status"`
	NowPlayingSource    string `json:"now_playing_source"`     // REST API 用 now_playing_source
	A2dpSourceName      string `json:"a2dp_source_name"`       // 蓝牙源名称
	AudioVolume         int    `json:"audio_volume"`           // REST API 用 audio_volume
	AudioVolumeIncrement int   `json:"audio_volume_increment"`
	AudioVolumeMax      int    `json:"audio_volume_max"`
	NowPlayingTitle     string `json:"now_playing_title"`
	NowPlayingArtist    string `json:"now_playing_artist"`
	NowPlayingAlbum     string `json:"now_playing_album"`
	NowPlayingDuration  int    `json:"now_playing_duration"`
	NowPlayingElapsed   int    `json:"now_playing_elapsed"`
	NowPlayingStation   string `json:"now_playing_station"`
}

// VehicleData 完整车辆数据
// 注意：Tesla API 的 vehicle_data 返回格式中，state 字段在 response 外面
// 但有时 API 返回缓存数据时 state 可能为空字符串
// 所以必须通过 deriveVehicleState() 推导真实状态
type VehicleData struct {
	Response VehicleDataResponse `json:"response"`
	State    string              `json:"state"` // online/asleep/offline，可能为空
}

type RealtimeData struct {
	Speed                    float64 `json:"speed"`
	Gear                     string  `json:"gear"`
	Power                    float64 `json:"power"`
	PedalPosition            float64 `json:"pedal_position"`
	CruiseSetSpeed           float64 `json:"cruise_set_speed"`
	LateralAcceleration      float64 `json:"lateral_acceleration"`
	LongitudinalAcceleration float64 `json:"longitudinal_acceleration"`
	Latitude                 float64 `json:"latitude"`
	Longitude                float64 `json:"longitude"`
	Heading                  int     `json:"heading"`
	GpsState                 int     `json:"gps_state"`
	Soc                      float64 `json:"soc"`
	BatteryLevel             float64 `json:"battery_level"`
	DCChargingPower          float64 `json:"dc_charging_power"`
	ACChargingPower          float64 `json:"ac_charging_power"`
	PackVoltage              float64 `json:"pack_voltage"`
	PackCurrent              float64 `json:"pack_current"`
	EnergyRemaining          float64 `json:"energy_remaining"`
	ChargeAmps               float64 `json:"charge_amps"`
	ChargerVoltage           float64 `json:"charger_voltage"`
	ChargeState              string  `json:"charge_state"`
	FastChargerPresent       bool    `json:"fast_charger_present"`
	UpdatedAt                int64   `json:"updated_at"`
}

type VehicleStateData struct {
	Locked             bool    `json:"locked"`
	DoorOpen           bool    `json:"door_open"`
	DoorFL             bool    `json:"door_fl"`
	DoorFR             bool    `json:"door_fr"`
	DoorRL             bool    `json:"door_rl"`
	DoorRR             bool    `json:"door_rr"`
	TrunkOpen          bool    `json:"trunk_open"`
	FrunkOpen          bool    `json:"frunk_open"`
	WindowsOpen        bool    `json:"windows_open"`
	FdWindow           bool    `json:"fd_window"`
	FpWindow           bool    `json:"fp_window"`
	RdWindow           bool    `json:"rd_window"`
	RpWindow           bool    `json:"rp_window"`
	SentryMode         bool    `json:"sentry_mode"`
	ValetModeEnabled   bool    `json:"valet_mode_enabled"`
	ServiceMode        bool    `json:"service_mode"`
	InsideTemp         float64 `json:"inside_temp"`
	OutsideTemp        float64 `json:"outside_temp"`
	DriverTempSetting  float64 `json:"driver_temp_setting"`
	PassengerTempSetting float64 `json:"passenger_temp_setting"`
	IsACOn             bool    `json:"is_ac_on"`
	IsClimateOn        bool    `json:"is_climate_on"`
	SeatHeaterLeft     int     `json:"seat_heater_left"`
	SeatHeaterRight    int     `json:"seat_heater_right"`
	SeatHeaterRearLeft int     `json:"seat_heater_rear_left"`
	SeatHeaterRearRight int    `json:"seat_heater_rear_right"`
	SeatHeaterRearCenter int   `json:"seat_heater_rear_center"`
	SteeringWheelHeater bool   `json:"steering_wheel_heater"`
	DefrostMode        int     `json:"defrost_mode"`
	HvacPower          bool    `json:"hvac_power"`
	HvacACEnabled      bool    `json:"hvac_ac_enabled"`
	HvacAutoMode       bool    `json:"hvac_auto_mode"`
	HvacFanSpeed       int     `json:"hvac_fan_speed"`
	ClimateKeeperMode  int     `json:"climate_keeper_mode"`
	ChargePortDoorOpen bool    `json:"charge_port_door_open"`
	ChargePortLatch    string  `json:"charge_port_latch"`
	ChargeLimitSoc     int     `json:"charge_limit_soc"`
	MinutesToFull      int     `json:"minutes_to_full"`
	ChargingState      string  `json:"charging_state"`
	ChargeSpeed        float64 `json:"charge_speed"`
	Voltage            int     `json:"voltage"`
	Ampere             float64 `json:"ampere"`
	ChargePower        float64 `json:"charge_power"`
	AddedEnergy        float64 `json:"added_energy"`
	ChargeEnergyAdded  float64 `json:"charge_energy_added"`
	FastChargerType    string  `json:"fast_charger_type"`
	BatteryHeaterOn    bool    `json:"battery_heater_on"`
	TpmsFL             float64 `json:"tpms_fl"`
	TpmsFR             float64 `json:"tpms_fr"`
	TpmsRL             float64 `json:"tpms_rl"`
	TpmsRR             float64 `json:"tpms_rr"`
	CarType            string  `json:"car_type"`
	Trim               string  `json:"trim"`
	ExteriorColor      string  `json:"exterior_color"`
	WheelType          string  `json:"wheel_type"`
	Version            string  `json:"version"`
	OdometerKm         float64 `json:"odometer_km"`
	RangeKm            float64 `json:"range_km"`
	DriverSeatBelt     int     `json:"driver_seat_belt"`
	DriverSeatOccupied bool    `json:"driver_seat_occupied"`
	CenterDisplayState int     `json:"center_display_state"`
	MirrorFolded       bool    `json:"mirror_folded"`
	LightsHighBeams    bool    `json:"lights_high_beams"`
	LightsHazardsActive bool   `json:"lights_hazards_active"`
	LightsTurnSignal   string  `json:"lights_turn_signal"`
	BrakePedal         bool    `json:"brake_pedal"`
	DriveRail          bool    `json:"drive_rail"`
	PedalPosition      float64 `json:"pedal_position"`
	GuestModeEnabled   bool    `json:"guest_mode_enabled"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationName      string  `json:"destination_name"`
	BatteryTemp        float64 `json:"battery_temp"`
	CurrentLimitMph    float64 `json:"current_limit_mph"`
	UpdatedAt          int64   `json:"updated_at"`
}

type MediaStateData struct {
	PlaybackStatus       string `json:"media_playback_status"`
	AudioSource          string `json:"media_audio_source"`
	Volume               int    `json:"media_volume"`
	AudioVolumeIncrement int    `json:"media_audio_volume_increment"`
	AudioVolumeMax       int    `json:"media_audio_volume_max"`
	NowPlayingTitle      string `json:"now_playing_title"`
	NowPlayingArtist     string `json:"now_playing_artist"`
	NowPlayingAlbum      string `json:"now_playing_album"`
	NowPlayingDuration   int    `json:"now_playing_duration"`
	NowPlayingElapsed    int    `json:"now_playing_elapsed"`
	NowPlayingStation    string `json:"now_playing_station"`
	UpdatedAt            int64  `json:"updated_at"`
}

func ExtractRealtimeFromSimple(data *SimpleVehicleData) map[string]interface{} {
	m := map[string]interface{}{}
	if data.Speed != 0 {
		m["speed"] = data.Speed
	}
	if data.Gear != "" {
		m["gear"] = data.Gear
	}
	if data.Power != 0 {
		m["power"] = data.Power
	}
	if data.PedalPosition != 0 {
		m["pedal_position"] = data.PedalPosition
	}
	if data.Latitude != 0 || data.Longitude != 0 {
		m["latitude"] = data.Latitude
		m["longitude"] = data.Longitude
	}
	if data.Heading != 0 {
		m["heading"] = data.Heading
	}
	if data.Soc != 0 {
		m["soc"] = float64(data.Soc)
		m["battery_level"] = float64(data.Soc)
	}
	if data.UsableSoc != 0 {
		m["usable_soc"] = data.UsableSoc
	}
	if data.RangeKm != 0 {
		m["range_km"] = data.RangeKm
	}
	if data.BatteryTemp != 0 {
		m["battery_temp"] = data.BatteryTemp
	}
	if data.EnergyRemaining != 0 {
		m["energy_remaining"] = data.EnergyRemaining
	}
	if data.ChargePower != 0 {
		m["dc_charging_power"] = data.ChargePower
		m["charge_power"] = data.ChargePower
	}
	if data.DcChargingPower != 0 {
		m["dc_charging_power"] = data.DcChargingPower
	}
	if data.AcChargingPower != 0 {
		m["ac_charging_power"] = data.AcChargingPower
	}
	if data.Voltage != 0 {
		m["charger_voltage"] = float64(data.Voltage)
		m["voltage"] = data.Voltage
	}
	if data.ChargerVoltage != 0 {
		m["charger_voltage"] = data.ChargerVoltage
	}
	if data.Ampere != 0 {
		m["charge_amps"] = data.Ampere
		m["ampere"] = data.Ampere
	}
	if data.ChargeAmps != 0 {
		m["charge_amps"] = data.ChargeAmps
	}
	if data.PackVoltage != 0 {
		m["pack_voltage"] = data.PackVoltage
	}
	if data.PackCurrent != 0 {
		m["pack_current"] = data.PackCurrent
	}
	if data.ChargingState != "" {
		m["charge_state"] = data.ChargingState
		m["charging_state"] = data.ChargingState
	}
	if data.Supercharging {
		m["fast_charger_present"] = data.Supercharging
	}
	if data.FastChargerPresent {
		m["fast_charger_present"] = data.FastChargerPresent
	}
	if data.ChargeLimitSoc != 0 {
		m["charge_limit_soc"] = data.ChargeLimitSoc
	}
	if data.ChargeSpeed != 0 {
		m["charge_speed"] = data.ChargeSpeed
	}
	if data.AddedEnergy != 0 {
		m["added_energy"] = data.AddedEnergy
	}
	if data.ChargeEnergyAdded != 0 {
		m["charge_energy_added"] = data.ChargeEnergyAdded
	}
	if data.MinutesToFull != 0 {
		m["minutes_to_full"] = data.MinutesToFull
	}
	if data.TimeToFullCharge != 0 {
		m["time_to_full_charge"] = data.TimeToFullCharge
	}
	if data.ChargePortDoorOpen {
		m["charge_port_door_open"] = data.ChargePortDoorOpen
	}
	if data.ChargePortOpen {
		m["charge_port_open"] = data.ChargePortOpen
	}
	if data.ChargePortLatch != "" {
		m["charge_port_latch"] = data.ChargePortLatch
	}
	if data.ChargeCurrentRequest != 0 {
		m["charge_current_request"] = data.ChargeCurrentRequest
	}
	if data.ChargeCurrentRequestMax != 0 {
		m["charge_current_request_max"] = data.ChargeCurrentRequestMax
	}
	if data.DcChargingEnergyIn != 0 {
		m["dc_charging_energy_in"] = data.DcChargingEnergyIn
	}
	if data.AcChargingEnergyIn != 0 {
		m["ac_charging_energy_in"] = data.AcChargingEnergyIn
	}
	if data.ChargerPhases != 0 {
		m["charger_phases"] = data.ChargerPhases
	}
	if data.ModuleTempMax != 0 {
		m["module_temp_max"] = data.ModuleTempMax
	}
	if data.ModuleTempMin != 0 {
		m["module_temp_min"] = data.ModuleTempMin
	}
	if data.BrickVoltageMax != 0 {
		m["brick_voltage_max"] = data.BrickVoltageMax
	}
	if data.BrickVoltageMin != 0 {
		m["brick_voltage_min"] = data.BrickVoltageMin
	}
	if data.BatteryHeaterOn {
		m["battery_heater_on"] = data.BatteryHeaterOn
	}
	if data.BmsState != nil {
		m["bms_state"] = data.BmsState
	}
	if data.BmsFullChargeComplete {
		m["bms_full_charge_complete"] = data.BmsFullChargeComplete
	}
	if data.DcdcEnable {
		m["dcdc_enable"] = data.DcdcEnable
	}
	if data.IsolationResistance != 0 {
		m["isolation_resistance"] = data.IsolationResistance
	}
	if data.LifetimeEnergyUsed != 0 {
		m["lifetime_energy_used"] = data.LifetimeEnergyUsed
	}
	if data.PreconditioningEnabled {
		m["preconditioning_enabled"] = data.PreconditioningEnabled
	}
	m["updated_at"] = time.Now().UnixMilli()
	return m
}

func ExtractStateFromSimple(data *SimpleVehicleData) map[string]interface{} {
	m := map[string]interface{}{}
	// Fleet API 规则：
	// - 布尔字段：false 是合法值（如 locked=false, sentry_mode=false），必须推送
	// - 数值字段 0：需要区分语义
	//   - speed=0: Fleet API 返回 null 时 Go 解析为 0，null≠0，不推送
	//   - seat_heater=0: 合法值（0=关闭），必须推送
	//   - charger_voltage=0: 合法值（未充电），但不应覆盖遥测充电数据，不推送
	//   - defrost_mode=0: 合法值（0=关闭），必须推送
	// - 字符串字段 "": Fleet API 返回 null 时 Go 解析为 ""，null≠""，不推送

	// 布尔字段：直接推送，false 也是合法值
	m["locked"] = data.Locked
	m["door_open"] = data.DoorOpen
	m["door_fl"] = data.DoorFL
	m["door_fr"] = data.DoorFR
	m["door_rl"] = data.DoorRL
	m["door_rr"] = data.DoorRR
	m["trunk_open"] = data.TrunkOpen
	m["frunk_open"] = data.FrunkOpen
	m["windows_open"] = data.WindowsOpen
	m["fd_window"] = data.FdWindow
	m["fp_window"] = data.FpWindow
	m["rd_window"] = data.RdWindow
	m["rp_window"] = data.RpWindow
	m["sentry_mode"] = data.SentryMode
	m["service_mode"] = data.ServiceMode
	m["is_ac_on"] = data.IsACOn
	m["is_climate_on"] = data.IsClimateOn
	m["brake_pedal"] = data.BrakePedal
	m["drive_rail"] = data.DriveRail
	m["guest_mode_enabled"] = data.GuestModeEnabled
	m["charge_port_door_open"] = data.ChargePortDoorOpen
	m["battery_heater_on"] = data.BatteryHeaterOn
	m["hvac_ac_enabled"] = data.HvacACEnabled
	m["steering_wheel_heater"] = data.SteeringWheelHeater

	// 数值字段：0 是合法值（0=关闭/无），必须推送
	// 这些字段的 0 有明确语义：座椅加热0=关闭，除霜0=关闭，风扇0=关闭
	m["seat_heater_left"] = data.SeatHeaterLeft
	m["seat_heater_right"] = data.SeatHeaterRight
	m["seat_heater_rear_left"] = data.SeatHeaterRearLeft
	m["seat_heater_rear_right"] = data.SeatHeaterRearRight
	m["seat_heater_rear_center"] = data.SeatHeaterRearCenter
	m["defrost_mode"] = data.DefrostMode
	m["hvac_fan_speed"] = data.HvacFanSpeed
	m["charge_limit_soc"] = data.ChargeLimitSoc

	// 数值字段：0 可能是"无数据"也可能是"真实0"，需要根据上下文判断
	// 温度字段：0 可能是真实0度（极端天气），但更可能是传感器未初始化
	// Fleet API 在车辆休眠唤醒后可能返回 0.0 作为占位
	if data.InsideTemp != 0 {
		m["inside_temp"] = data.InsideTemp
	}
	if data.OutsideTemp != 0 {
		m["outside_temp"] = data.OutsideTemp
	}
	if data.DriverTempSetting != 0 {
		m["driver_temp_setting"] = data.DriverTempSetting
	}
	if data.PassengerTempSetting != 0 {
		m["passenger_temp_setting"] = data.PassengerTempSetting
	}
	if data.HvacPower {
		m["hvac_power"] = data.HvacPower
	}
	if data.HvacAutoMode {
		m["hvac_auto_mode"] = data.HvacAutoMode
	}
	if data.ClimateKeeperMode != 0 {
		m["climate_keeper_mode"] = data.ClimateKeeperMode
	}

	// 充电相关字段：0 是合法值（未充电时为0），但不应覆盖遥测的充电数据
	// Fleet API 返回完整的充电状态快照，0 值是真实的
	if data.ChargePortLatch != "" {
		m["charge_port_latch"] = data.ChargePortLatch
	}
	if data.ChargingState != "" {
		m["charging_state"] = data.ChargingState
	}
	if data.ChargeSpeed != 0 {
		m["charge_speed"] = data.ChargeSpeed
	}
	if data.Voltage != 0 {
		m["voltage"] = data.Voltage
	}
	if data.Ampere != 0 {
		m["ampere"] = data.Ampere
	}
	if data.ChargePower != 0 {
		m["charge_power"] = data.ChargePower
	}
	if data.AddedEnergy != 0 {
		m["added_energy"] = data.AddedEnergy
	}
	if data.ChargeEnergyAdded != 0 {
		m["charge_energy_added"] = data.ChargeEnergyAdded
	}
	if data.FastChargerType != "" {
		m["fast_charger_type"] = data.FastChargerType
	}
	if data.MinutesToFull != 0 {
		m["minutes_to_full"] = data.MinutesToFull
	}

	// 胎压：0 可能是传感器未初始化
	if data.TpmsFL != 0 {
		m["tpms_fl"] = data.TpmsFL
	}
	if data.TpmsFR != 0 {
		m["tpms_fr"] = data.TpmsFR
	}
	if data.TpmsRL != 0 {
		m["tpms_rl"] = data.TpmsRL
	}
	if data.TpmsRR != 0 {
		m["tpms_rr"] = data.TpmsRR
	}

	// 里程/版本等：0 通常不是合法值
	if data.CarType != "" {
		m["car_type"] = data.CarType
	}
	if data.ExteriorColor != "" {
		m["exterior_color"] = data.ExteriorColor
	}
	if data.Version != "" {
		m["version"] = data.Version
	}
	if data.OdometerKm != 0 {
		m["odometer_km"] = data.OdometerKm
	}
	if data.RangeKm != 0 {
		m["range_km"] = data.RangeKm
	}
	if data.CenterDisplayState != 0 {
		m["center_display_state"] = data.CenterDisplayState
	}
	if data.MirrorFolded {
		m["mirror_folded"] = data.MirrorFolded
	}
	if data.LightsHighBeams {
		m["lights_high_beams"] = data.LightsHighBeams
	}
	if data.LightsHazardsActive {
		m["lights_hazards_active"] = data.LightsHazardsActive
	}
	if data.LightsTurnSignal != "" {
		m["lights_turn_signal"] = data.LightsTurnSignal
	}
	if data.PedalPosition != 0 {
		m["pedal_position"] = data.PedalPosition
	}
	if data.DestinationLatitude != 0 || data.DestinationLongitude != 0 {
		m["destination_latitude"] = data.DestinationLatitude
		m["destination_longitude"] = data.DestinationLongitude
	}
	if data.DestinationName != "" {
		m["destination_name"] = data.DestinationName
	}
	if data.BatteryTemp != 0 {
		m["battery_temp"] = data.BatteryTemp
	}
	if data.CurrentLimitMph != 0 {
		m["current_limit_mph"] = data.CurrentLimitMph
	}

	// 遥测扩展字段：从 SimpleVehicleData 提取，确保 WS state_update 事件包含这些字段
	if data.RatedRangeKm != 0 {
		m["rated_range_km"] = data.RatedRangeKm
	}
	if data.WheelType != "" {
		m["wheel_type"] = data.WheelType
	}
	if data.RoofColor != "" {
		m["roof_color"] = data.RoofColor
	}
	if data.Trim != "" {
		m["trim"] = data.Trim
	}
	if data.EfficiencyPackage != "" {
		m["efficiency_package"] = data.EfficiencyPackage
	}
	if data.BatteryLevel != 0 {
		m["battery_level"] = data.BatteryLevel
	}
	if data.EnergyRemaining != 0 {
		m["energy_remaining"] = data.EnergyRemaining
	}
	if data.TimeToFullCharge != 0 {
		m["time_to_full_charge"] = data.TimeToFullCharge
	}
	if data.DcChargingPower != 0 {
		m["dc_charging_power"] = data.DcChargingPower
	}
	if data.AcChargingPower != 0 {
		m["ac_charging_power"] = data.AcChargingPower
	}
	if data.ChargeAmps != 0 {
		m["charge_amps"] = data.ChargeAmps
	}
	if data.ChargerVoltage != 0 {
		m["charger_voltage"] = data.ChargerVoltage
	}
	if data.FastChargerPresent {
		m["fast_charger_present"] = data.FastChargerPresent
	}
	if data.ChargePortOpen {
		m["charge_port_open"] = data.ChargePortOpen
	}
	if data.ChargeCurrentRequest != 0 {
		m["charge_current_request"] = data.ChargeCurrentRequest
	}
	if data.ChargeCurrentRequestMax != 0 {
		m["charge_current_request_max"] = data.ChargeCurrentRequestMax
	}
	if data.DcChargingEnergyIn != 0 {
		m["dc_charging_energy_in"] = data.DcChargingEnergyIn
	}
	if data.AcChargingEnergyIn != 0 {
		m["ac_charging_energy_in"] = data.AcChargingEnergyIn
	}
	if data.ChargePortColdWeatherMode {
		m["charge_port_cold_weather_mode"] = data.ChargePortColdWeatherMode
	}
	if data.ChargeEnableRequest {
		m["charge_enable_request"] = data.ChargeEnableRequest
	}
	if data.ModuleTempMax != 0 {
		m["module_temp_max"] = data.ModuleTempMax
	}
	if data.ModuleTempMin != 0 {
		m["module_temp_min"] = data.ModuleTempMin
	}
	if data.NumModuleTempMax != 0 {
		m["num_module_temp_max"] = data.NumModuleTempMax
	}
	if data.NumModuleTempMin != 0 {
		m["num_module_temp_min"] = data.NumModuleTempMin
	}
	if data.BrickVoltageMax != 0 {
		m["brick_voltage_max"] = data.BrickVoltageMax
	}
	if data.BrickVoltageMin != 0 {
		m["brick_voltage_min"] = data.BrickVoltageMin
	}
	if data.NumBrickVoltageMax != 0 {
		m["num_brick_voltage_max"] = data.NumBrickVoltageMax
	}
	if data.NumBrickVoltageMin != 0 {
		m["num_brick_voltage_min"] = data.NumBrickVoltageMin
	}
	if data.BmsState != nil {
		m["bms_state"] = data.BmsState
	}
	if data.BmsFullChargeComplete {
		m["bms_full_charge_complete"] = data.BmsFullChargeComplete
	}
	if data.DcdcEnable {
		m["dcdc_enable"] = data.DcdcEnable
	}
	if data.IsolationResistance != 0 {
		m["isolation_resistance"] = data.IsolationResistance
	}
	if data.LifetimeEnergyUsed != 0 {
		m["lifetime_energy_used"] = data.LifetimeEnergyUsed
	}
	if data.PreconditioningEnabled {
		m["preconditioning_enabled"] = data.PreconditioningEnabled
	}
	if data.NotEnoughPowerToHeat {
		m["not_enough_power_to_heat"] = data.NotEnoughPowerToHeat
	}
	if data.Hvil {
		m["hvil"] = data.Hvil
	}
	if data.ChargingCableType != "" {
		m["charging_cable_type"] = data.ChargingCableType
	}
	if data.PackVoltage != 0 {
		m["pack_voltage"] = data.PackVoltage
	}
	if data.PackCurrent != 0 {
		m["pack_current"] = data.PackCurrent
	}
	if data.VehicleName != "" {
		m["vehicle_name"] = data.VehicleName
	}

	m["updated_at"] = time.Now().UnixMilli()
	return m
}

func ExtractMediaFromSimple(data *SimpleVehicleData) map[string]interface{} {
	m := map[string]interface{}{}
	if data.MediaPlaybackStatus != "" {
		m["media_playback_status"] = data.MediaPlaybackStatus
	}
	if data.MediaAudioSource != "" {
		m["media_audio_source"] = data.MediaAudioSource
	}
	if data.MediaVolume != 0 {
		m["media_volume"] = data.MediaVolume
	}
	if data.MediaAudioVolumeIncrement != 0 {
		m["media_audio_volume_increment"] = data.MediaAudioVolumeIncrement
	}
	if data.MediaAudioVolumeMax != 0 {
		m["media_audio_volume_max"] = data.MediaAudioVolumeMax
	}
	if data.NowPlayingTitle != "" {
		m["now_playing_title"] = data.NowPlayingTitle
	}
	if data.NowPlayingArtist != "" {
		m["now_playing_artist"] = data.NowPlayingArtist
	}
	if data.NowPlayingAlbum != "" {
		m["now_playing_album"] = data.NowPlayingAlbum
	}
	if data.NowPlayingDuration != 0 {
		m["now_playing_duration"] = data.NowPlayingDuration
	}
	if data.NowPlayingElapsed != 0 {
		m["now_playing_elapsed"] = data.NowPlayingElapsed
	}
	if data.NowPlayingStation != "" {
		m["now_playing_station"] = data.NowPlayingStation
	}
	m["updated_at"] = time.Now().UnixMilli()
	return m
}

// SimpleVehicleData 简化车辆数据结构，用于前端展示
// 字段命名遵循 Tesla Fleet API 前端字段映射规范
// 规范原则：Tesla原始字段 → 后端标准字段 → 前端UI字段
type SimpleVehicleData struct {
	ID        uint64    `json:"id"`
	VIN       string    `json:"vin"`
	UpdatedAt time.Time `json:"updated_at"`
	Online    bool      `json:"online"`
	State     string    `json:"state"`
	Driving   bool      `json:"driving"`
	Charging  bool      `json:"charging"`
	Soc       int       `json:"soc,omitempty"`
	UsableSoc int       `json:"usable_soc,omitempty"`
	RangeKm   float64   `json:"range_km,omitempty"`
	OdometerKm        float64 `json:"odometer_km,omitempty"`
	ChargingState     string  `json:"charging_state,omitempty"`
	ChargeSpeed       float64 `json:"charge_speed,omitempty"`
	ChargePower       float64 `json:"charge_power,omitempty"`
	Ampere            float64 `json:"ampere,omitempty"`
	Voltage           int     `json:"voltage,omitempty"`
	AddedEnergy       float64 `json:"added_energy,omitempty"`
	MinutesToFull     int     `json:"minutes_to_full,omitempty"`
	ChargeLimitSoc    int     `json:"charge_limit_soc"`
	Supercharging     bool    `json:"supercharging"`
	Gear     string  `json:"gear,omitempty"`
	Speed    float64 `json:"speed,omitempty"`
	Power    float64 `json:"power,omitempty"`
	Heading  int     `json:"heading,omitempty"`
	Latitude float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Locked      bool `json:"locked"`
	SentryMode  bool `json:"sentry_mode"`
	MirrorFolded   bool    `json:"mirror_folded"`
	WindowsOpen    bool    `json:"windows_open"`
	DoorOpen       bool    `json:"door_open"`
	TrunkOpen      bool    `json:"trunk_open"`
	FrunkOpen      bool    `json:"frunk_open"`
	ChargePortDoorOpen bool    `json:"charge_port_door_open"`
	ChargePortLatch    string  `json:"charge_port_latch,omitempty"`
	DoorFL bool `json:"door_fl"`
	DoorFR bool `json:"door_fr"`
	DoorRL bool `json:"door_rl"`
	DoorRR bool `json:"door_rr"`
	FdWindow bool `json:"fd_window"`
	FpWindow bool `json:"fp_window"`
	RdWindow bool `json:"rd_window"`
	RpWindow bool `json:"rp_window"`
	InsideTemp  float64 `json:"inside_temp,omitempty"`
	OutsideTemp float64 `json:"outside_temp,omitempty"`
	DriverTempSetting    float64 `json:"driver_temp_setting,omitempty"`
	PassengerTempSetting float64 `json:"passenger_temp_setting,omitempty"`
	IsACOn       bool  `json:"is_ac_on"`
	IsClimateOn  bool  `json:"is_climate_on"`
	Version      string `json:"version,omitempty"`
	TpmsFL float64 `json:"tpms_fl,omitempty"`
	TpmsFR float64 `json:"tpms_fr,omitempty"`
	TpmsRL float64 `json:"tpms_rl,omitempty"`
	TpmsRR float64 `json:"tpms_rr,omitempty"`
	BatteryTemp    float64 `json:"battery_temp,omitempty"`
	ChargerPhases  int     `json:"charger_phases,omitempty"`
	Lightweight    bool    `json:"lightweight"`
	MediaPlaybackStatus     string `json:"media_playback_status,omitempty"`
	MediaAudioSource        string `json:"media_audio_source,omitempty"`
	MediaVolume             int    `json:"media_volume,omitempty"`
	MediaAudioVolumeIncrement int  `json:"media_audio_volume_increment,omitempty"`
	MediaAudioVolumeMax     int    `json:"media_audio_volume_max,omitempty"`
	NowPlayingTitle         string `json:"now_playing_title,omitempty"`
	NowPlayingArtist    string `json:"now_playing_artist,omitempty"`
	NowPlayingAlbum     string `json:"now_playing_album,omitempty"`
	NowPlayingStation   string `json:"now_playing_station,omitempty"`
	NowPlayingDuration  int    `json:"now_playing_duration,omitempty"`
	NowPlayingElapsed   int    `json:"now_playing_elapsed,omitempty"`
	CenterDisplayState  int    `json:"center_display_state,omitempty"`
	StateOutput    *state.VehicleStateOutput `json:"state_output,omitempty"`
	SeatHeaterLeft      int     `json:"seat_heater_left"`
	SeatHeaterRight     int     `json:"seat_heater_right"`
	SeatHeaterRearLeft  int     `json:"seat_heater_rear_left"`
	SeatHeaterRearRight int     `json:"seat_heater_rear_right"`
	SeatHeaterRearCenter int    `json:"seat_heater_rear_center"`
	SteeringWheelHeater bool    `json:"steering_wheel_heater"`
	DefrostMode         int     `json:"defrost_mode"`
	HvacPower           bool    `json:"hvac_power,omitempty"`
	HvacACEnabled       bool    `json:"hvac_ac_enabled"`
	HvacAutoMode        bool    `json:"hvac_auto_mode"`
	HvacFanSpeed        int     `json:"hvac_fan_speed"`
	ClimateKeeperMode   int     `json:"climate_keeper_mode,omitempty"`
	ChargeEnergyAdded   float64 `json:"charge_energy_added,omitempty"`
	FastChargerType     string  `json:"fast_charger_type,omitempty"`
	BatteryHeaterOn     bool    `json:"battery_heater_on"`
	CarType             string  `json:"car_type,omitempty"`
	ExteriorColor       string  `json:"exterior_color,omitempty"`
	LightsHighBeams     bool    `json:"lights_high_beams"`
	LightsHazardsActive bool    `json:"lights_hazards_active"`
	LightsTurnSignal    string  `json:"lights_turn_signal,omitempty"`
	BrakePedal          bool    `json:"brake_pedal"`
	DriveRail           bool    `json:"drive_rail"`
	PedalPosition       float64 `json:"pedal_position,omitempty"`
	GuestModeEnabled    bool    `json:"guest_mode_enabled"`
	ServiceMode         bool    `json:"service_mode"`
	DestinationLatitude      float64 `json:"destination_latitude,omitempty"`
	DestinationLongitude     float64 `json:"destination_longitude,omitempty"`
	DestinationName          string  `json:"destination_name,omitempty"`
	MilesToArrival           float64 `json:"miles_to_arrival,omitempty"`
	MinutesToArrival         float64 `json:"minutes_to_arrival,omitempty"`
	CurrentLimitMph          float64 `json:"current_limit_mph,omitempty"`
	CruiseState              string  `json:"cruise_state,omitempty"`
	AutosteerState           string  `json:"autosteer_state,omitempty"`
	CruiseControlState       string  `json:"cruise_control_state,omitempty"`
	LaneKeepingState         string  `json:"lane_keeping_state,omitempty"`

	// 遥测扩展字段（Fleet API 不返回，由遥测写入 Redis，omitempty 防止 Fleet API 保存时覆盖遥测值）
	RatedRangeKm            float64 `json:"rated_range_km,omitempty"`
	WheelType               string  `json:"wheel_type,omitempty"`
	RoofColor               string  `json:"roof_color,omitempty"`
	Trim                    string  `json:"trim,omitempty"`
	EfficiencyPackage       string  `json:"efficiency_package,omitempty"`
	BatteryLevel            float64 `json:"battery_level,omitempty"`
	EnergyRemaining         float64 `json:"energy_remaining,omitempty"`
	TimeToFullCharge        float64 `json:"time_to_full_charge,omitempty"`
	DcChargingPower         float64 `json:"dc_charging_power,omitempty"`
	AcChargingPower         float64 `json:"ac_charging_power,omitempty"`
	ChargeAmps              float64 `json:"charge_amps,omitempty"`
	ChargerVoltage          float64 `json:"charger_voltage,omitempty"`
	FastChargerPresent      bool    `json:"fast_charger_present,omitempty"`
	ChargePortOpen          bool    `json:"charge_port_open,omitempty"`
	ChargeCurrentRequest    int     `json:"charge_current_request,omitempty"`
	ChargeCurrentRequestMax int     `json:"charge_current_request_max,omitempty"`
	DcChargingEnergyIn      float64 `json:"dc_charging_energy_in,omitempty"`
	AcChargingEnergyIn      float64 `json:"ac_charging_energy_in,omitempty"`
	ChargePortColdWeatherMode bool  `json:"charge_port_cold_weather_mode,omitempty"`
	ChargeEnableRequest     bool    `json:"charge_enable_request,omitempty"`
	ModuleTempMax           float64 `json:"module_temp_max,omitempty"`
	ModuleTempMin           float64 `json:"module_temp_min,omitempty"`
	NumModuleTempMax        int     `json:"num_module_temp_max,omitempty"`
	NumModuleTempMin        int     `json:"num_module_temp_min,omitempty"`
	BrickVoltageMax         float64 `json:"brick_voltage_max,omitempty"`
	BrickVoltageMin         float64 `json:"brick_voltage_min,omitempty"`
	NumBrickVoltageMax      int     `json:"num_brick_voltage_max,omitempty"`
	NumBrickVoltageMin      int     `json:"num_brick_voltage_min,omitempty"`
	BmsState                interface{} `json:"bms_state,omitempty"` // 遥测可能返回 string 或 int
	BmsFullChargeComplete   bool    `json:"bms_full_charge_complete,omitempty"`
	DcdcEnable              bool    `json:"dcdc_enable,omitempty"`
	IsolationResistance     float64 `json:"isolation_resistance,omitempty"`
	LifetimeEnergyUsed      float64 `json:"lifetime_energy_used,omitempty"`
	PreconditioningEnabled  bool    `json:"preconditioning_enabled,omitempty"`
	NotEnoughPowerToHeat    bool    `json:"not_enough_power_to_heat,omitempty"`
	Hvil                    bool    `json:"hvil,omitempty"`
	ChargingCableType       string  `json:"charging_cable_type,omitempty"`
	PackVoltage             float64 `json:"pack_voltage,omitempty"`
	PackCurrent             float64 `json:"pack_current,omitempty"`
	VehicleName             string  `json:"vehicle_name,omitempty"`
}

// VirtualKeyStatus 虚拟钥匙状态
type VirtualKeyStatus struct {
	KeyPaired       bool   `json:"key_paired"`
	KeyCount        int    `json:"key_count"`
	CommandRequired bool   `json:"command_protocol_required"`
	SignedCommand   bool   `json:"signed_command_available"`
	FleetTelemetry  string `json:"fleet_telemetry_version"`
	DiscountedData  bool   `json:"discounted_device_data"`
}

// TokenResponse Token 响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// CommandResponse 命令响应（包含错误处理）
type CommandResponse struct {
	Response struct {
		Result bool   `json:"result"`
		Reason string `json:"reason"`
	} `json:"response"`
	Error string `json:"error"`
}

// VehicleInfo 车辆信息
type VehicleInfo struct {
	ID          int64  `json:"id"`
	IDS         string `json:"id_s"` // 关键：Fleet API 真正使用的是 id_s（字符串）
	VehicleID   int64  `json:"vehicle_id"`
	VIN         string `json:"vin"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
	InService   bool   `json:"in_service"`
	AccessType  string `json:"access_type"`
	APIVersion  int    `json:"api_version"`
}

// VehicleInfoResponse 车辆信息响应
type VehicleInfoResponse struct {
	Response VehicleInfo `json:"response"`
}

// FleetStatusRequest 车队状态请求
type FleetStatusRequest struct {
	VINs []string `json:"vins"`
}

// FleetStatusVehicleInfo 车队状态车辆信息
type FleetStatusVehicleInfo struct {
	FirmwareVersion                string `json:"firmware_version"`
	VehicleCommandProtocolRequired bool   `json:"vehicle_command_protocol_required"`
	DiscountedDeviceData           bool   `json:"discounted_device_data"`
	FleetTelemetryVersion          string `json:"fleet_telemetry_version"`
	TotalNumberOfKeys              int    `json:"total_number_of_keys"`
	SafetyScreenStreamingToggle    *bool  `json:"safety_screen_streaming_toggle_enabled"`
}

// FleetStatusResponse 车队状态响应
type FleetStatusResponse struct {
	Response struct {
		KeyPairedVINs []string                          `json:"key_paired_vins"`
		UnpairedVINs  []string                          `json:"unpaired_vins"`
		VehicleInfo   map[string]FleetStatusVehicleInfo `json:"vehicle_info"`
	} `json:"response"`
}

// GetVehicleData 获取完整车辆数据
// GET /api/1/vehicles/{vehicle_tag}/vehicle_data
// 推荐 endpoints（中国区稳定）：location_data;charge_state;drive_state;vehicle_state;climate_state;gui_settings;vehicle_config
// 注意：fleet 层只接受 vehicleTag，不做 VIN 查询
func GetVehicleData(accessToken, vehicleTag string) (*VehicleData, error) {
	return getVehicleDataWithRetry(accessToken, vehicleTag, 0)
}

func getVehicleDataWithRetry(accessToken, vehicleTag string, retryCount int) (*VehicleData, error) {
	cfg := config.Load()
	// 使用推荐的 endpoints（移除 closures_state，中国区不支持）
	url := fmt.Sprintf("%s/api/1/vehicles/%s/vehicle_data?endpoints=location_data%%3Bcharge_state%%3Bdrive_state%%3Bvehicle_state%%3Bclimate_state%%3Bgui_settings%%3Bvehicle_config%%3Bmedia_state%%3Bmedia_info",
		cfg.Tesla.FleetAPIURL, vehicleTag)

	log.Printf("[Fleet API] Fetching vehicle_data (vehicle_tag: %s)", vehicleTag)

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		Get(url)

	if err != nil {
		return nil, err
	}

	log.Printf("[Fleet API] vehicle_data status: %d", resp.StatusCode())

	body := resp.Body()

	if resp.StatusCode() == 200 {
		var errResp TeslaError
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			log.Printf("[Fleet API Error] %s", errResp.Error)
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}

		var data VehicleData
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("failed to parse vehicle data: %v", err)
		}

		if data.Response.MediaInfo.NowPlayingTitle != "" {
			log.Printf("[Fleet API] media_info received: title=%q artist=%q source=%q",
				data.Response.MediaInfo.NowPlayingTitle,
				data.Response.MediaInfo.NowPlayingArtist,
				data.Response.MediaInfo.NowPlayingSource)
		}

		return &data, nil
	}

	if resp.StatusCode() == 500 {
		var errResp TeslaError
		if err := json.Unmarshal(body, &errResp); err == nil {
			log.Printf("[Fleet API Error 500] %s", errResp.Error)

			if strings.Contains(errResp.Error, "vehicle not found") {
				log.Printf("[Fleet API] Vehicle not found - check vehicle_tag and authorization")
				return nil, fmt.Errorf("vehicle not found: %s", errResp.Error)
			}

			// BUG 6 修复：删除自动唤醒逻辑
			// 睡眠是正常状态，不应该自动 wake_up
			// 只有用户主动操作时才允许唤醒
			if strings.Contains(errResp.Error, "vehicle unavailable") ||
				strings.Contains(errResp.Error, "vehicle is offline") ||
				strings.Contains(errResp.Error, "vehicle is asleep") {
				return nil, fmt.Errorf("vehicle asleep or offline: %s", errResp.Error)
			}
		}
	}

	// BUG 6 修复：408 也不自动唤醒
	if resp.StatusCode() == 408 {
		return nil, fmt.Errorf("vehicle timeout (may be asleep): %d", resp.StatusCode())
	}

	var errResp TeslaError
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		log.Printf("[Fleet API Error] %s", errResp.Error)
		return nil, fmt.Errorf("API error: %s", errResp.Error)
	}
	return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(body))
}

// WakeUp 唤醒车辆
// POST /api/1/vehicles/{vehicle_tag}/wake_up
// 限制：同一车辆 5分钟最多 1~3 次 wake_up
// BUG 3 修复：改为同步执行，避免异步导致的时序混乱
func WakeUp(accessToken, vehicleTag string) error {
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles/%s/wake_up", cfg.Tesla.FleetAPIURL, vehicleTag)

	log.Printf("[WakeUp] Sending wake_up request (vehicle_tag: %s)", vehicleTag)

	maxAttempts := 2
	backoff := []time.Duration{2, 5}

	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			delay := backoff[i-1] * time.Second
			log.Printf("[WakeUp] Retrying wake_up (attempt %d/%d), waiting %ds before retry...", i+1, maxAttempts, delay/time.Second)
			time.Sleep(delay)
		}

		startTime := time.Now()
		resp, err := wakeUpClient.R().
			SetHeader("Authorization", "Bearer "+accessToken).
			SetHeader("Content-Type", "application/json").
			Post(url)
		elapsed := time.Since(startTime)

		if err != nil {
			log.Printf("[WakeUp] Attempt %d failed in %v: %v", i+1, elapsed, err)
			if strings.Contains(err.Error(), "context deadline exceeded") {
				log.Printf("[WakeUp WARNING] Timeout waiting for Tesla Fleet API response")
			}
			continue
		}

		log.Printf("[WakeUp] Attempt %d completed in %v with status %d", i+1, elapsed, resp.StatusCode())

		if resp.StatusCode() == 200 {
			log.Printf("[WakeUp] Successfully woke up vehicle")
			return nil
		}
	}

	return fmt.Errorf("wake_up failed after %d attempts", maxAttempts)
}

// deriveVehicleState 根据车辆数据推导业务状态
// 状态优先级（从高到低）：
//   driving:  speed > 0 || shift_state in (D, R)
//   charging: charging_state == "Charging"
//   online:   state == "online"
//   asleep:   state == "asleep" || (有缓存数据但 state 为空)
//   offline:  真正的离线（无数据或 API 错误）
//
// 🔥 关键：Tesla 的 asleep 是在线状态的一种，只是 MCU 休眠
// 不是断网，不应该显示为 offline
//
// 完整状态列表（按优先级排序）：
//   driving    - 行驶中 (speed > 0 || shift_state in D/R)
//   charging   - 充电中 (charging_state == "Charging")
//   updating   - 软件更新中 (vehicle_state.software_update_status)
//   climate_on - 空调运行中 (is_climate_on && !driving && !charging)
//   sentry_on  - 哨兵模式 (sentry_mode && !driving && !charging)
//   online     - 在线待机 (state == "online")
//   waking     - 唤醒中 (state 从 asleep 变为 online 的过渡)
//   asleep     - 睡眠 (state == "asleep" || 有缓存数据但 state 为空)
//   offline    - 离线 (无数据或 API 错误)
func deriveVehicleState(data *VehicleData) string {
	// 1. 行驶中：速度 > 0 或档位在 D/R（优先级最高）
	if data.Response.DriveState.Speed > 0 ||
		data.Response.DriveState.ShiftState == "D" ||
		data.Response.DriveState.ShiftState == "R" {
		return "driving"
	}

	// 2. 充电中
	if data.Response.ChargeState.ChargingState == "Charging" {
		return "charging"
	}

	// 3. 软件更新中
	if data.Response.VehicleState.CenterDisplayState == 3 || // 更新显示状态
		(data.Response.VehicleState.CarVersion != "" && data.State == "updating") {
		return "updating"
	}

	// 4. 空调运行中（停车状态）
	if data.Response.ClimateState.IsClimateOn &&
		data.Response.DriveState.ShiftState == "P" &&
		data.Response.ChargeState.ChargingState != "Charging" {
		return "climate_on"
	}

	// 5. 哨兵模式（停车状态）
	if data.Response.VehicleState.SentryMode &&
		data.Response.DriveState.ShiftState == "P" &&
		data.Response.ChargeState.ChargingState != "Charging" {
		return "sentry_on"
	}

	// 6. 在线（停车但唤醒）
	if data.State == "online" {
		return "online"
	}

	// 7. 睡眠 - 明确标记为 asleep
	if data.State == "asleep" {
		return "asleep"
	}

	// 🔥 关键修复：处理 Tesla API 的 "假离线" 情况
	// 当 online=false, state="" 但仍有有效数据时，
	// 说明这是缓存数据，车辆实际上处于睡眠状态
	// 而不是真正的离线
	hasValidData := data.Response.ChargeState.BatteryLevel > 0 ||
		data.Response.VehicleState.Odometer > 0 ||
		data.Response.DriveState.Latitude != 0

	// BUG 1 修复：删除 !Locked 条件，因为 Tesla 睡眠时 locked 可能为 true
	if hasValidData && data.State == "" {
		return "asleep"
	}

	// 8. 离线 - 真正的离线（无数据或 API 错误）
	return "offline"
}

// deriveHvacPower 根据 ClimateState 推导 HVAC 功率（kW）
// Tesla API 不直接提供 HVAC 功率，根据空调状态估算
func deriveHvacPower(cs ClimateState) float64 {
	if !cs.IsClimateOn {
		return 0
	}
	// 粗略估算：空调开启时约 1-3kW，PTC 加热时可达 5-6kW
	power := 1.0
	if cs.DefrostMode > 0 {
		power += 2.0
	}
	if cs.SteeringWheelHeat {
		power += 0.3
	}
	if cs.SeatHeaterLeft > 0 || cs.SeatHeaterRight > 0 {
		power += 0.2
	}
	return power
}

// GetVehicleState 获取车辆状态（简化数据）
func GetVehicleState(accessToken, vehicleTag string) (*SimpleVehicleData, error) {
	data, err := GetVehicleData(accessToken, vehicleTag)
	if err != nil {
		log.Printf("[Fleet API] GetVehicleData failed: %v", err)
		return nil, err
	}

	// 推导业务状态（不是直接使用 API 的 state 字段）
	derivedState := deriveVehicleState(data)
	// BUG 2 修复：asleep 是在线状态的一种，只是 MCU 休眠，不是断网
	// 所以只有 offline 才是真正的离线
	isOnline := derivedState != "offline"
	isDriving := derivedState == "driving"

	// 单位转换：统一将英里转换为公里
	// 注意：不要依赖 gui_distance_units，Tesla API 的显示单位和底层单位经常不一致
	// 即使 gui_distance_units = "km/hr"，API 返回的里程数据可能仍然是英里
	odometerKm := milesToKm(data.Response.VehicleState.Odometer)
	batteryRangeKm := milesToKm(data.Response.ChargeState.BatteryRange)

	// speed 需要转换：Tesla API 返回的是 mph（英里/小时），需要转换为 km/h（公里/小时）
	speed := milesToKm(float64(data.Response.DriveState.Speed))

	gcjLat, gcjLng := geo.WGS84ToGCJ02(data.Response.DriveState.Latitude, data.Response.DriveState.Longitude)

	return &SimpleVehicleData{
		ID:        1,
		VIN:       "",
		UpdatedAt: time.Now(),
		Online:      isOnline,
		State:       derivedState,
		Driving:     isDriving,
		Charging:    data.Response.ChargeState.ChargingState == "Charging",
		Soc:         data.Response.ChargeState.BatteryLevel,
		UsableSoc:   data.Response.ChargeState.UsableBatteryLevel,
		RangeKm:     batteryRangeKm,
		OdometerKm:  odometerKm,
		ChargingState:     data.Response.ChargeState.ChargingState,
		ChargeSpeed:       milesToKm(data.Response.ChargeState.ChargeRate),
		ChargePower:       float64(data.Response.ChargeState.ChargerPower),
		Ampere:            float64(data.Response.ChargeState.ChargerCurrent),
		Voltage:           data.Response.ChargeState.ChargerVoltage,
		AddedEnergy:       data.Response.ChargeState.ChargeEnergyAdded,
		MinutesToFull:     data.Response.ChargeState.MinutesToFullCharge,
		ChargeLimitSoc:    data.Response.ChargeState.ChargeLimitSoc,
		Supercharging:     data.Response.ChargeState.FastChargerPresent,
		Gear:     data.Response.DriveState.ShiftState,
		Speed:    speed,
		Power:    float64(data.Response.DriveState.Power),
		Heading:  data.Response.DriveState.Heading,
		Latitude: gcjLat,
		Longitude: gcjLng,
		Locked:      data.Response.VehicleState.Locked,
		SentryMode:  data.Response.VehicleState.SentryMode,
		MirrorFolded:   data.Response.VehicleState.MirrorFolded,
		// REST API 返回 int 类型：df/pf/dr/pr (0:关, 1:开), ft/rt (0:关, 1或2:开/解锁)
		// 用 >0 判断，避免尾门运动中(值为2)被误判为关闭
		DoorFL: data.Response.VehicleState.DF > 0,
		DoorFR: data.Response.VehicleState.PF > 0,
		DoorRL: data.Response.VehicleState.DR > 0,
		DoorRR: data.Response.VehicleState.PR > 0,
		DoorOpen:    data.Response.VehicleState.DF > 0 || data.Response.VehicleState.PF > 0 || data.Response.VehicleState.DR > 0 || data.Response.VehicleState.PR > 0,
		TrunkOpen:   data.Response.VehicleState.RT > 0,
		FrunkOpen:   data.Response.VehicleState.FT > 0,
		// REST API 返回 int 类型：fd_window/fp_window/rd_window/rp_window (0:关, >0:开)
		// 数值代表电机位移，只要不等于0就视为未关严
		FdWindow: data.Response.VehicleState.FdWindow > 0,
		FpWindow: data.Response.VehicleState.FpWindow > 0,
		RdWindow: data.Response.VehicleState.RdWindow > 0,
		RpWindow: data.Response.VehicleState.RpWindow > 0,
		WindowsOpen: data.Response.VehicleState.FdWindow > 0 || data.Response.VehicleState.FpWindow > 0 || data.Response.VehicleState.RdWindow > 0 || data.Response.VehicleState.RpWindow > 0,
		ChargePortDoorOpen: data.Response.ChargeState.ChargePortOpen,
		ChargePortLatch:    data.Response.ChargeState.ChargePortLatch,
		InsideTemp:           data.Response.ClimateState.InsideTemp,
		OutsideTemp:          data.Response.ClimateState.OutsideTemp,
		DriverTempSetting:    data.Response.ClimateState.DriverTempSetting,
		PassengerTempSetting: data.Response.ClimateState.PassengerTempSetting,
		IsACOn:      data.Response.ClimateState.IsClimateOn,
		IsClimateOn: data.Response.ClimateState.IsClimateOn,
		SeatHeaterLeft:      data.Response.ClimateState.SeatHeaterLeft,
		SeatHeaterRight:     data.Response.ClimateState.SeatHeaterRight,
		SeatHeaterRearLeft:  data.Response.ClimateState.SeatHeaterRearLeft,
		SeatHeaterRearRight: data.Response.ClimateState.SeatHeaterRearRight,
		SeatHeaterRearCenter: data.Response.ClimateState.SeatHeaterRearCenter,
		SteeringWheelHeater: data.Response.ClimateState.SteeringWheelHeat,
		DefrostMode:         int(data.Response.ClimateState.DefrostMode),
		HvacPower:           data.Response.ClimateState.IsClimateOn,
		HvacACEnabled:       data.Response.ClimateState.IsAirConditioningOn,
		HvacAutoMode:        data.Response.ClimateState.AutoConditioningEnabled,
		HvacFanSpeed:        data.Response.ClimateState.FanStatus,
		ClimateKeeperMode:   int(data.Response.ClimateState.ClimateKeeperMode),
		Version:     data.Response.VehicleState.CarVersion,
		TpmsFL: data.Response.VehicleState.TPMSPressureFL,
		TpmsFR: data.Response.VehicleState.TPMSPressureFR,
		TpmsRL: data.Response.VehicleState.TPMSPressureRL,
		TpmsRR: data.Response.VehicleState.TPMSPressureRR,
		BatteryTemp:   data.Response.ClimateState.BatteryTemp,
		ChargerPhases: 0,
		Lightweight:   false,
		ChargeEnergyAdded:  data.Response.ChargeState.ChargeEnergyAdded,
		FastChargerType:    data.Response.ChargeState.FastChargerType,
		BatteryHeaterOn:    data.Response.ChargeState.BatteryHeaterOn,
		CarType:            data.Response.VehicleConfig.CarType,
		ExteriorColor:      data.Response.VehicleConfig.ExteriorColor,
		LightsHighBeams:    data.Response.DriveState.LightsHighBeams,
		LightsHazardsActive: data.Response.VehicleState.LightsHazardsActive,
		LightsTurnSignal:   data.Response.DriveState.LightsTurnSignal,
		BrakePedal:         data.Response.DriveState.BrakePedal,
		DriveRail:          data.Response.DriveState.DriveRail,
		PedalPosition:      data.Response.DriveState.AcceleratorPedalPos,
		GuestModeEnabled:   data.Response.VehicleState.GuestModeEnabled,
		ServiceMode:        data.Response.VehicleState.ServiceMode,
		DestinationLatitude:      data.Response.DriveState.ActiveRouteLatitude,
		DestinationLongitude:     data.Response.DriveState.ActiveRouteLongitude,
		DestinationName:          data.Response.DriveState.ActiveRouteName,
		MilesToArrival:           data.Response.DriveState.ActiveRouteMilesToArrival,
		MinutesToArrival:         data.Response.DriveState.ActiveRouteMinutesToArrival,
		CurrentLimitMph:          data.Response.VehicleState.SpeedLimitMode.CurrentLimitMph,
		CruiseState:              data.Response.DriveState.CruiseState,
		AutosteerState:           data.Response.DriveState.AutosteerState,
		CruiseControlState:       data.Response.DriveState.CruiseControlState,
		LaneKeepingState:         data.Response.DriveState.LaneKeepingState,
		MediaPlaybackStatus:     data.Response.MediaInfo.MediaPlaybackStatus,
		MediaAudioSource:        data.Response.MediaInfo.NowPlayingSource,
		MediaVolume:             data.Response.MediaInfo.AudioVolume,
		MediaAudioVolumeIncrement: data.Response.MediaInfo.AudioVolumeIncrement,
		MediaAudioVolumeMax:     data.Response.MediaInfo.AudioVolumeMax,
		NowPlayingTitle:         data.Response.MediaInfo.NowPlayingTitle,
		NowPlayingArtist:        data.Response.MediaInfo.NowPlayingArtist,
		NowPlayingAlbum:         data.Response.MediaInfo.NowPlayingAlbum,
		NowPlayingStation:       data.Response.MediaInfo.NowPlayingStation,
		NowPlayingDuration:      data.Response.MediaInfo.NowPlayingDuration,
		NowPlayingElapsed:       data.Response.MediaInfo.NowPlayingElapsed,
		CenterDisplayState:  data.Response.VehicleState.CenterDisplayState,
		// 遥测别名字段：Fleet API 返回的数据映射到前端期望的字段名
		BatteryLevel:       float64(data.Response.ChargeState.BatteryLevel),
		ChargerVoltage:     float64(data.Response.ChargeState.ChargerVoltage),
		ChargeAmps:         float64(data.Response.ChargeState.ChargerCurrent),
		FastChargerPresent: data.Response.ChargeState.FastChargerPresent,
		ChargePortOpen:     data.Response.ChargeState.ChargePortOpen,
	}, nil
}

// isVehicleOnline 判断车辆是否在线
func isVehicleOnline(data *VehicleData) bool {
	if data.State == "online" {
		return true
	}

	// center_display_state 在 vehicle_state 下
	if data.Response.VehicleState.CenterDisplayState != 0 {
		return true
	}

	// shift_state 在 D/R 表示行驶中
	if data.Response.DriveState.ShiftState == "D" ||
		data.Response.DriveState.ShiftState == "R" {
		return true
	}

	if data.Response.ChargeState.ChargingState == "Charging" {
		return true
	}

	if data.Response.DriveState.Speed > 0 {
		return true
	}

	return false
}

// SendCommand 通过 VCP 代理发送签名命令
// VCP(签名命令)必须使用 VIN 作为车辆标识符，不能使用 id_s
// 当车辆进入 VehicleCommandProtocolRequired 模式后，Fleet API 直连 command 也会 403，因此不回退
// 限制：最低间隔 2 秒
func SendCommand(accessToken, vin, command string, body interface{}) (*CommandResponse, error) {
	cfg := config.Load()

	if cfg.Tesla.VCPURL == "" {
		return nil, fmt.Errorf("VCP URL not configured, signed command required but no proxy available")
	}

	return sendCommandViaVCP(cfg.Tesla.VCPURL, accessToken, vin, command, body)
}

// sendCommandViaVCP 通过 Vehicle Command Proxy 发送签名命令
// tesla-http-proxy API: POST https://host:port/api/1/vehicles/{VIN}/command/{command}
// VCP 必须使用 VIN 作为车辆标识符，不能使用 id_s
// Body: 命令参数直接放在body中（如 {"on": true}），无命令时传 {}
func sendCommandViaVCP(vcpURL, accessToken, vin, command string, body interface{}) (*CommandResponse, error) {
	url := fmt.Sprintf("%s/api/1/vehicles/%s/command/%s", strings.TrimRight(vcpURL, "/"), vin, command)

	log.Printf("[VCP] Sending signed command '%s' via VCP (vin: %s)", command, vin)

	reqBody := body
	if reqBody == nil {
		reqBody = map[string]interface{}{}
	}

	resp, err := vcpClient.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(url)

	if err != nil {
		log.Printf("[VCP] Command '%s' failed: %v", command, err)
		return nil, fmt.Errorf("VCP command failed: %v", err)
	}

	log.Printf("[VCP] Command '%s' response status: %d, body: %s", command, resp.StatusCode(), string(resp.Body()))

	var cmdResp CommandResponse
	if err := json.Unmarshal(resp.Body(), &cmdResp); err != nil {
		if resp.StatusCode() == 200 {
			return &CommandResponse{
				Response: struct {
					Result bool   `json:"result"`
					Reason string `json:"reason"`
				}{
					Result: true,
					Reason: "",
				},
				Error: "",
			}, nil
		}
		return nil, fmt.Errorf("failed to parse VCP response: %v", err)
	}

	if cmdResp.Error != "" {
		return &cmdResp, fmt.Errorf("VCP command error: %s", cmdResp.Error)
	}

	return &cmdResp, nil
}

// GetVehicleInfo 获取车辆信息
// GET /api/1/vehicles/{vehicle_tag}
func GetVehicleInfo(accessToken, vehicleTag string) (*VehicleInfo, error) {
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles/%s", cfg.Tesla.FleetAPIURL, vehicleTag)

	log.Printf("[Fleet API] Getting vehicle info (vehicle_tag: %s)", vehicleTag)

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		Get(url)

	if err != nil {
		return nil, err
	}

	log.Printf("[Fleet API] vehicle info status: %d", resp.StatusCode())

	if resp.StatusCode() == 200 {
		var data VehicleInfoResponse
		if err := json.Unmarshal(resp.Body(), &data); err != nil {
			log.Printf("[Fleet API] Failed to parse vehicle info response: %v", err)
			return nil, err
		}
		return &data.Response, nil
	}

	return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
}

// GetVehicles 获取所有车辆列表
// GET /api/1/vehicles
// 🔥 关键：此接口不会唤醒车辆，适合在睡眠模式下使用
func GetVehicles(accessToken string) ([]VehicleInfo, error) {
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles", cfg.Tesla.FleetAPIURL)

	log.Printf("[Fleet API] Getting vehicles list")

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		Get(url)

	if err != nil {
		return nil, err
	}

	log.Printf("[Fleet API] vehicles list status: %d", resp.StatusCode())

	if resp.StatusCode() == 200 {
		var data struct {
			Response []VehicleInfo `json:"response"`
		}
		if err := json.Unmarshal(resp.Body(), &data); err != nil {
			log.Printf("[Fleet API] Failed to parse vehicles list response: %v", err)
			return nil, err
		}
		return data.Response, nil
	}

	return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
}

// GetVehicleStateLightweight 轻量级获取车辆状态（不会唤醒车辆）
// 使用 /vehicles 接口获取基本状态，适合睡眠模式下的轮询
func GetVehicleStateLightweight(accessToken, vehicleTag string) (*SimpleVehicleData, error) {
	vehicles, err := GetVehicles(accessToken)
	if err != nil {
		return nil, err
	}

	// 查找对应车辆
	for _, v := range vehicles {
		if v.IDS == vehicleTag {
			return &SimpleVehicleData{
				ID:          uint64(v.ID),
				VIN:         v.VIN,
				UpdatedAt:   time.Now(),
				Online:      v.State != "offline",
				State:       v.State,
				Driving:     false,
				Charging:    false,
				Lightweight: true,
			}, nil
		}
	}

	return nil, fmt.Errorf("vehicle not found in list")
}

// CheckFleetStatus 检查车队状态
// POST /api/1/vehicles/fleet_status
func CheckFleetStatus(accessToken string, vins []string) (*FleetStatusResponse, error) {
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles/fleet_status", cfg.Tesla.FleetAPIURL)

	log.Printf("[Fleet API] Checking fleet status for %d vehicles", len(vins))

	reqBody := FleetStatusRequest{VINs: vins}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(url)

	if err != nil {
		return nil, err
	}

	log.Printf("[Fleet API] fleet_status status: %d", resp.StatusCode())

	if resp.StatusCode() == 200 {
		var data FleetStatusResponse
		if err := json.Unmarshal(resp.Body(), &data); err != nil {
			log.Printf("[Fleet API] Failed to parse fleet status response: %v", err)
			return nil, err
		}
		return &data, nil
	}

	return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
}

// GetFleetTelemetryErrors 查询车辆遥测连接错误（Tesla 官方推荐的排查手段）
func GetFleetTelemetryErrors(accessToken string, vin string) (interface{}, error) {
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles/%s/fleet_telemetry_errors", cfg.Tesla.FleetAPIURL, vin)

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		Get(url)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 200 {
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &result); err != nil {
			return nil, err
		}
		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			return nil, fmt.Errorf("Tesla API error: %s", errMsg)
		}
		return result, nil
	}

	return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode(), string(resp.Body()))
}

// VerifyVirtualKey 验证虚拟钥匙状态
func VerifyVirtualKey(accessToken string, vins []string) (*VirtualKeyStatus, error) {
	fleetStatus, err := CheckFleetStatus(accessToken, vins)
	if err != nil {
		return nil, err
	}

	keyPaired := false
	for _, pairedVIN := range fleetStatus.Response.KeyPairedVINs {
		for _, vin := range vins {
			if pairedVIN == vin {
				keyPaired = true
				break
			}
		}
	}

	result := &VirtualKeyStatus{
		KeyPaired:       keyPaired,
		KeyCount:        0,
		CommandRequired: true,
		SignedCommand:   true,
		FleetTelemetry:  "",
		DiscountedData:  false,
	}

	for _, vin := range vins {
		if vehicleInfo, hasInfo := fleetStatus.Response.VehicleInfo[vin]; hasInfo {
			result.CommandRequired = vehicleInfo.VehicleCommandProtocolRequired
			result.FleetTelemetry = vehicleInfo.FleetTelemetryVersion
			result.DiscountedData = vehicleInfo.DiscountedDeviceData
			result.KeyCount = vehicleInfo.TotalNumberOfKeys
			break
		}
	}

	return result, nil
}

type TelemetryConfigRequest struct {
	VINs    []string                `json:"vins"`
	Config  TelemetryConfigPayload  `json:"config"`
}

type TelemetryConfigPayload struct {
	Hostname string                    `json:"hostname"`
	Port     int                       `json:"port"`
	Fields   map[string]TelemetryField `json:"fields"`
	CA       string                    `json:"ca,omitempty"` // 服务器 CA 证书（PEM 格式）
}

type TelemetryField struct {
	IntervalSeconds int `json:"interval_seconds,omitempty"`
	MinimumDelta    int `json:"minimum_delta,omitempty"` // 某些字段要求必须设置，如 SelfDrivingMilesSinceReset
}

type TelemetryConfigResponse struct {
	Response struct {
		SuccessfulVINs  []string `json:"successful_vins,omitempty"`
		SkippedVINs     []string `json:"skipped_vins,omitempty"`
		UpdatedVehicles int      `json:"updated_vehicles,omitempty"`
	} `json:"response"`
	Error string `json:"error,omitempty"`
}

func ConfigureFleetTelemetry(accessToken string, vins []string, hostname string, caCertFile string) (*TelemetryConfigResponse, error) {
	cfg := config.Load()

	// 检查 VCP 是否配置
	if cfg.Tesla.VCPURL == "" {
		return nil, fmt.Errorf("VCP URL not configured, fleet telemetry config requires VCP proxy")
	}

	// 通过 VCP 代理发送
	url := fmt.Sprintf("%s/api/1/vehicles/fleet_telemetry_config", strings.TrimRight(cfg.Tesla.VCPURL, "/"))

	// Tesla Fleet Telemetry 配置字段
	// 官方机制：车辆每 500ms 刷新一次数据桶，将所有已发射的字段值推送到服务器
	// 每个字段只在 interval_seconds 已过 且 值已变化 时才发射到数据桶
	// interval_seconds 设为 1，Tesla 服务端会根据字段类型自动使用最小间隔
	// 例如：VehicleSpeed 最小 1s，BatteryLevel 最小 10s，TpmsPressure 最小 5s
	// 字段名必须来自官方文档：https://developer.tesla.cn/docs/fleet-api/fleet-telemetry/available-data
	telemetryFields := map[string]TelemetryField{
		// 媒体数据 (Media)
		"MediaNowPlayingTitle":    {IntervalSeconds: 1},
		"MediaNowPlayingArtist":   {IntervalSeconds: 1},
		"MediaNowPlayingAlbum":    {IntervalSeconds: 1},
		"MediaNowPlayingDuration": {IntervalSeconds: 1},
		"MediaNowPlayingElapsed":  {IntervalSeconds: 1},
		"MediaNowPlayingStation":  {IntervalSeconds: 1},
		"MediaPlaybackSource":     {IntervalSeconds: 1},
		"MediaPlaybackStatus":     {IntervalSeconds: 1},
		"MediaAudioVolume":        {IntervalSeconds: 1},
		"MediaAudioVolumeIncrement": {IntervalSeconds: 1},
		"MediaAudioVolumeMax":     {IntervalSeconds: 1},

		// 行驶数据 (Driving)
		"VehicleSpeed":             {IntervalSeconds: 1},
		"Gear":                     {IntervalSeconds: 1},
		"DriveRail":                {IntervalSeconds: 1},
		"PedalPosition":            {IntervalSeconds: 1},
		"BrakePedal":               {IntervalSeconds: 1},
		"BrakePedalPos":            {IntervalSeconds: 1},
		"LateralAcceleration":      {IntervalSeconds: 1},
		"LongitudinalAcceleration": {IntervalSeconds: 1},
		"DiStateF":                 {IntervalSeconds: 1},
		"DiStateR":                 {IntervalSeconds: 1},
		"DiTorqueActualF":          {IntervalSeconds: 1},
		"DiTorqueActualR":          {IntervalSeconds: 1},
		"DiAxleSpeedF":             {IntervalSeconds: 1},
		"DiAxleSpeedR":             {IntervalSeconds: 1},
		"DiStatorTempF":            {IntervalSeconds: 1},
		"DiStatorTempR":            {IntervalSeconds: 1},
		"DiHeatsinkTF":             {IntervalSeconds: 1},
		"DiHeatsinkTR":             {IntervalSeconds: 1},
		"DiInverterTF":             {IntervalSeconds: 1},
		"DiInverterTR":             {IntervalSeconds: 1},
		"DiMotorCurrentF":          {IntervalSeconds: 1},
		"DiMotorCurrentR":          {IntervalSeconds: 1},
		"DiVBatF":                  {IntervalSeconds: 1},
		"DiVBatR":                  {IntervalSeconds: 1},
		"DiSlaveTorqueCmd":         {IntervalSeconds: 1},
		"DiTorquemotor":            {IntervalSeconds: 1},
		"LifetimeEnergyUsedDrive":  {IntervalSeconds: 1},

		// 充电数据 (Charging)
		"BatteryLevel":        {IntervalSeconds: 1},
		"Soc":                 {IntervalSeconds: 1},
		"EnergyRemaining":     {IntervalSeconds: 1},
		"EstBatteryRange":     {IntervalSeconds: 1},
		"IdealBatteryRange":   {IntervalSeconds: 1},
		"RatedRange":          {IntervalSeconds: 1},
		"ChargeState":         {IntervalSeconds: 1},
		"DetailedChargeState": {IntervalSeconds: 1},
		"ChargeLimitSoc":      {IntervalSeconds: 1},
		"TimeToFullCharge":    {IntervalSeconds: 1},
		"EstimatedHoursToChargeTermination": {IntervalSeconds: 1},
		"PackCurrent":         {IntervalSeconds: 1},
		"PackVoltage":         {IntervalSeconds: 1},
		"DCChargingPower":     {IntervalSeconds: 1},
		"DCChargingEnergyIn":  {IntervalSeconds: 1},
		"ACChargingPower":     {IntervalSeconds: 1},
		"ACChargingEnergyIn":  {IntervalSeconds: 1},
		"FastChargerPresent":  {IntervalSeconds: 1},
		"FastChargerType":     {IntervalSeconds: 1},
		"ChargerVoltage":      {IntervalSeconds: 1},
		"ChargeAmps":          {IntervalSeconds: 1},
		"ChargeCurrentRequest":    {IntervalSeconds: 1},
		"ChargeCurrentRequestMax": {IntervalSeconds: 1},
		"ChargerPhases":           {IntervalSeconds: 1},
		"ChargingCableType":       {IntervalSeconds: 1},
		"ChargePortDoorOpen":      {IntervalSeconds: 1},
		"ChargePortLatch":         {IntervalSeconds: 1},
		"ChargePortColdWeatherMode": {IntervalSeconds: 1},
		"ChargeRateMilePerHour":   {IntervalSeconds: 1},
		"BatteryHeaterOn":         {IntervalSeconds: 1},
		"PreconditioningEnabled":  {IntervalSeconds: 1},
		"BMSState":                {IntervalSeconds: 1},
		"BmsFullchargecomplete":   {IntervalSeconds: 1},
		"DCDCEnable":              {IntervalSeconds: 1},
		"LifetimeEnergyUsed":      {IntervalSeconds: 1},
		"ModuleTempMax":           {IntervalSeconds: 1},
		"ModuleTempMin":           {IntervalSeconds: 1},
		"NumModuleTempMax":        {IntervalSeconds: 1},
		"NumModuleTempMin":        {IntervalSeconds: 1},
		"BrickVoltageMax":         {IntervalSeconds: 1},
		"BrickVoltageMin":         {IntervalSeconds: 1},
		"NumBrickVoltageMax":      {IntervalSeconds: 1},
		"NumBrickVoltageMin":      {IntervalSeconds: 1},
		"NotEnoughPowerToHeat":    {IntervalSeconds: 1},
		"SuperchargerSessionTripPlanner": {IntervalSeconds: 1},

		// Powershare 数据
		"PowershareStatus":              {IntervalSeconds: 1},
		"PowershareType":                {IntervalSeconds: 1},
		"PowershareInstantaneousPowerKW": {IntervalSeconds: 1},
		"PowershareHoursLeft":           {IntervalSeconds: 1},
		"PowershareStopReason":          {IntervalSeconds: 1},

		// 空调数据 (Climate)
		"InsideTemp":                   {IntervalSeconds: 1},
		"OutsideTemp":                  {IntervalSeconds: 1},
		"HvacPower":                    {IntervalSeconds: 1},
		"HvacACEnabled":                {IntervalSeconds: 1},
		"HvacAutoMode":                 {IntervalSeconds: 1},
		"HvacFanSpeed":                 {IntervalSeconds: 1},
		"HvacFanStatus":                {IntervalSeconds: 1},
		"HvacLeftTemperatureRequest":   {IntervalSeconds: 1},
		"HvacRightTemperatureRequest":  {IntervalSeconds: 1},
		"ClimateKeeperMode":            {IntervalSeconds: 1},
		"CabinOverheatProtectionMode":  {IntervalSeconds: 1},
		"CabinOverheatProtectionTemperatureLimit": {IntervalSeconds: 1},
		"DefrostMode":                  {IntervalSeconds: 1},
		"DefrostForPreconditioning":    {IntervalSeconds: 1},
		"HvacSteeringWheelHeatLevel":   {IntervalSeconds: 1},
		"HvacSteeringWheelHeatAuto":    {IntervalSeconds: 1},
		"SeatHeaterLeft":               {IntervalSeconds: 1},
		"SeatHeaterRight":              {IntervalSeconds: 1},
		"SeatHeaterRearLeft":           {IntervalSeconds: 1},
		"SeatHeaterRearRight":          {IntervalSeconds: 1},
		"SeatHeaterRearCenter":         {IntervalSeconds: 1},
		"ClimateSeatCoolingFrontLeft":  {IntervalSeconds: 1},
		"ClimateSeatCoolingFrontRight": {IntervalSeconds: 1},
		"AutoSeatClimateLeft":          {IntervalSeconds: 1},
		"AutoSeatClimateRight":         {IntervalSeconds: 1},
		"SeatVentEnabled":              {IntervalSeconds: 1},
		"RearDisplayHvacEnabled":       {IntervalSeconds: 1},
		"WiperHeatEnabled":             {IntervalSeconds: 1},

		// 定位数据 (Location)
		"Location":            {IntervalSeconds: 1},
		"GpsHeading":          {IntervalSeconds: 1},
		"GpsState":            {IntervalSeconds: 1},
		"DestinationLocation": {IntervalSeconds: 1},
		"DestinationName":     {IntervalSeconds: 1},
		"OriginLocation":      {IntervalSeconds: 1},
		"RouteLine":           {IntervalSeconds: 1},
		"RouteTrafficMinutesDelay": {IntervalSeconds: 1},
		"MilesToArrival":     {IntervalSeconds: 1},
		"MinutesToArrival":   {IntervalSeconds: 1},
		"ExpectedEnergyPercentAtTripArrival": {IntervalSeconds: 1},
		"LocatedAtHome":      {IntervalSeconds: 1},
		"LocatedAtWork":      {IntervalSeconds: 1},
		"LocatedAtFavorite":  {IntervalSeconds: 1},
		"RouteLastUpdated":   {IntervalSeconds: 1},

		// 车辆状态 (Vehicle State)
		"Locked":        {IntervalSeconds: 1},
		"DoorState":     {IntervalSeconds: 1},
		"FdWindow":      {IntervalSeconds: 1},
		"FpWindow":      {IntervalSeconds: 1},
		"RdWindow":      {IntervalSeconds: 1},
		"RpWindow":      {IntervalSeconds: 1},
		"Odometer":      {IntervalSeconds: 1},
		"CenterDisplay": {IntervalSeconds: 1},
		"SentryMode":    {IntervalSeconds: 1},
		"ServiceMode":   {IntervalSeconds: 1},
		"GuestModeEnabled":         {IntervalSeconds: 1},
		"GuestModeMobileAccessState": {IntervalSeconds: 1},
		"HomelinkDeviceCount":      {IntervalSeconds: 1},
		"HomelinkNearby":           {IntervalSeconds: 1},
		"LightsHazardsActive":      {IntervalSeconds: 1},
		"LightsHighBeams":          {IntervalSeconds: 1},
		"LightsTurnSignal":         {IntervalSeconds: 1},
		"DriverSeatOccupied":       {IntervalSeconds: 1},
		"DriverSeatBelt":           {IntervalSeconds: 1},
		"PassengerSeatBelt":        {IntervalSeconds: 1},
		"PairedPhoneKeyAndKeyFobQty": {IntervalSeconds: 1},
		"PinToDriveEnabled":        {IntervalSeconds: 1},

		// 安全/ADAS 数据
		"AutomaticBlindSpotCamera":       {IntervalSeconds: 1},
		"BlindSpotCollisionWarningChime": {IntervalSeconds: 1},
		"ForwardCollisionWarning":        {IntervalSeconds: 1},
		"LaneDepartureAvoidance":         {IntervalSeconds: 1},
		"EmergencyLaneDepartureAvoidance": {IntervalSeconds: 1},
		"AutomaticEmergencyBrakingOff":   {IntervalSeconds: 1},
		"CurrentLimitMph":                {IntervalSeconds: 1},
		"MilesSinceReset":                {IntervalSeconds: 1, MinimumDelta: 1},
		"SelfDrivingMilesSinceReset":     {IntervalSeconds: 1, MinimumDelta: 1},

		// 胎压 (TPMS)
		"TpmsPressureFl": {IntervalSeconds: 5},
		"TpmsPressureFr": {IntervalSeconds: 5},
		"TpmsPressureRl": {IntervalSeconds: 5},
		"TpmsPressureRr": {IntervalSeconds: 5},
		"TpmsLastSeenPressureTimeFl": {IntervalSeconds: 5},
		"TpmsLastSeenPressureTimeFr": {IntervalSeconds: 5},
		"TpmsLastSeenPressureTimeRl": {IntervalSeconds: 5},
		"TpmsLastSeenPressureTimeRr": {IntervalSeconds: 5},
		"TpmsSoftWarnings":    {IntervalSeconds: 5},
		"TpmsHardWarnings":    {IntervalSeconds: 5},
		"IsolationResistance": {IntervalSeconds: 5},

		// 车辆配置 (Vehicle Config)
		"CarType":               {IntervalSeconds: 10},
		"Trim":                  {IntervalSeconds: 10},
		"VehicleName":           {IntervalSeconds: 10},
		"Version":               {IntervalSeconds: 10},
		"ExteriorColor":         {IntervalSeconds: 10},
		"RoofColor":             {IntervalSeconds: 10},
		"WheelType":             {IntervalSeconds: 10},
		"EuropeVehicle":         {IntervalSeconds: 10},
		"RightHandDrive":        {IntervalSeconds: 10},
		"EfficiencyPackage":     {IntervalSeconds: 10},
		"RearSeatHeaters":       {IntervalSeconds: 10},
		"SunroofInstalled":      {IntervalSeconds: 10},
		"RemoteStartEnabled":    {IntervalSeconds: 10},
		"Setting24HourTime":     {IntervalSeconds: 10},
		"SettingChargeUnit":     {IntervalSeconds: 10},
		"SettingDistanceUnit":   {IntervalSeconds: 10},
		"SettingTemperatureUnit": {IntervalSeconds: 10},
		"SettingTirePressureUnit": {IntervalSeconds: 10},

		// 软件更新 (Software Update)
		"SoftwareUpdateVersion":                      {IntervalSeconds: 10},
		"SoftwareUpdateDownloadPercentComplete":       {IntervalSeconds: 10},
		"SoftwareUpdateExpectedDurationMinutes":       {IntervalSeconds: 10},
		"SoftwareUpdateInstallationPercentComplete":   {IntervalSeconds: 10},
		"SoftwareUpdateScheduledStartTime":            {IntervalSeconds: 10},

		// Cybertruck
		"TonneauPosition":      {IntervalSeconds: 1},
		"TonneauOpenPercent":   {IntervalSeconds: 1},
		"TonneauTentMode":      {IntervalSeconds: 1},
		"OffroadLightbarPresent": {IntervalSeconds: 1},
	}

	// 读取 CA 证书（如果有）
	var caCertPEM string
	if caCertFile != "" {
		caData, err := os.ReadFile(caCertFile)
		if err != nil {
			log.Printf("[VCP] Warning: failed to read CA cert file %s: %v", caCertFile, err)
		} else {
			caCertPEM = string(caData)
			log.Printf("[VCP] Loaded CA cert from %s", caCertFile)
		}
	}

	// 解析端口号（默认 443）
	port := 443
	if cfg.Telemetry.ListenAddr != "" && cfg.Telemetry.ListenAddr[0] == ':' {
		if p, err := strconv.Atoi(cfg.Telemetry.ListenAddr[1:]); err == nil {
			port = p
		}
	}

	reqBody := TelemetryConfigRequest{
		VINs: vins,
		Config: TelemetryConfigPayload{
			Hostname: hostname,
			Port:     port,
			Fields:   telemetryFields,
			CA:       caCertPEM,
		},
	}

	log.Printf("[VCP] Configuring Fleet Telemetry for %d vehicles, hostname=%s", len(vins), hostname)

	resp, err := vcpClient.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("fleet telemetry config request via VCP failed: %w", err)
	}

	log.Printf("[VCP] fleet_telemetry_config status: %d", resp.StatusCode())
	log.Printf("[VCP] fleet_telemetry_config response: %s", string(resp.Body()))

	var result TelemetryConfigResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse telemetry config response: %w", err)
	}

	if result.Error != "" {
		return &result, fmt.Errorf("fleet telemetry config error: %s", result.Error)
	}

	log.Printf("[VCP] Telemetry config: successful=%v, skipped=%v, updated_vehicles=%d",
		result.Response.SuccessfulVINs, result.Response.SkippedVINs, result.Response.UpdatedVehicles)

	return &result, nil
}
