package fleet

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
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
	DoorFL             bool    `json:"door_fl"`
	DoorFR             bool    `json:"door_fr"`
	DoorRL             bool    `json:"door_rl"`
	DoorRR             bool    `json:"door_rr"`
	TrunkOpen          bool    `json:"trunk_open"`
	FrunkOpen          bool    `json:"frunk_open"`
	WindowFL           bool    `json:"window_fl"`
	WindowFR           bool    `json:"window_fr"`
	WindowRL           bool    `json:"window_rl"`
	WindowRR           bool    `json:"window_rr"`
	Locked             bool    `json:"locked"`
	SentryMode         bool    `json:"sentry_mode"`
	CarVersion         string  `json:"car_version"`
	CenterDisplayState int     `json:"center_display_state"`
	MirrorFolded       bool    `json:"mirror_folded"`
	TPMSPressureFL float64 `json:"tpms_pressure_fl"`
	TPMSPressureFR float64 `json:"tpms_pressure_fr"`
	TPMSPressureRL float64 `json:"tpms_pressure_rl"`
	TPMSPressureRR float64 `json:"tpms_pressure_rr"`
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
	ChargeLimitSoc      int     `json:"charge_limit_soc"`
	OutsideTemp         float64 `json:"outside_temp"`
}

// DriveState 驾驶状态
type DriveState struct {
	ShiftState string  `json:"shift_state"`
	Speed      int     `json:"speed"`
	Power      int     `json:"power"`
	Heading    int     `json:"heading"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

// ClimateState 气候状态
type ClimateState struct {
	InsideTemp           float64 `json:"inside_temp"`
	OutsideTemp          float64 `json:"outside_temp"`
	DriverTempSetting    float64 `json:"driver_temp_setting"`
	PassengerTempSetting float64 `json:"passenger_temp_setting"`
	IsClimateOn          bool    `json:"is_climate_on"`
	BatteryTemp          float64 `json:"battery_temp"`
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
	// 注意：closures_state 在中国区很多账号不支持，已移除
}

// MediaState 媒体状态（REST API vehicle_data 的 media_state 端点）
// 注意：REST API 的 media_state 只返回 remote_control_enabled
// 丰富的媒体数据（now_playing 等）通过 Fleet Telemetry 的 media_info 类别推送
type MediaState struct {
	RemoteControlEnabled bool `json:"remote_control_enabled"`
}

// MediaInfo 媒体信息（Fleet Telemetry media_info 类别）
// REST API 的 vehicle_data 可能也支持 media_info 端点返回这些数据
type MediaInfo struct {
	PlaybackStatus  string `json:"media_playback_status"`
	AudioSource     string `json:"media_audio_source"`
	Volume          int    `json:"media_volume"`
	NowPlayingTitle  string `json:"now_playing_title"`
	NowPlayingArtist string `json:"now_playing_artist"`
	NowPlayingAlbum  string `json:"now_playing_album"`
	NowPlayingDuration int  `json:"now_playing_duration"`
	NowPlayingElapsed  int  `json:"now_playing_elapsed"`
}

// VehicleData 完整车辆数据
// 注意：Tesla API 的 vehicle_data 返回格式中，state 字段在 response 外面
// 但有时 API 返回缓存数据时 state 可能为空字符串
// 所以必须通过 deriveVehicleState() 推导真实状态
type VehicleData struct {
	Response VehicleDataResponse `json:"response"`
	State    string              `json:"state"` // online/asleep/offline，可能为空
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
	Soc       int       `json:"soc"`
	UsableSoc int       `json:"usable_soc"`
	RangeKm   float64   `json:"range_km"`
	OdometerKm        float64 `json:"odometer_km"`
	ChargingState     string  `json:"charging_state"`
	ChargeSpeed       float64 `json:"charge_speed"`
	ChargePower       float64 `json:"charge_power"`
	Ampere            float64 `json:"ampere"`
	Voltage           int     `json:"voltage"`
	AddedEnergy       float64 `json:"added_energy"`
	MinutesToFull     int     `json:"minutes_to_full"`
	ChargeLimitSoc    int     `json:"charge_limit_soc"`
	Supercharging     bool    `json:"supercharging"`
	Gear     string  `json:"gear"`
	Speed    float64 `json:"speed"`
	Power    float64 `json:"power"`
	Heading  int     `json:"heading"`
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Locked      bool `json:"locked"`
	SentryMode  bool `json:"sentry_mode"`
	MirrorFolded   bool    `json:"mirror_folded"`
	WindowsOpen    bool    `json:"windows_open"`
	DoorOpen       bool    `json:"door_open"`
	TrunkOpen      bool    `json:"trunk_open"`
	FrunkOpen      bool    `json:"frunk_open"`
	ChargePortOpen bool    `json:"charge_port_open"`
	DoorFL bool `json:"door_fl"`
	DoorFR bool `json:"door_fr"`
	DoorRL bool `json:"door_rl"`
	DoorRR bool `json:"door_rr"`
	WindowFL bool `json:"window_fl"`
	WindowFR bool `json:"window_fr"`
	WindowRL bool `json:"window_rl"`
	WindowRR bool `json:"window_rr"`
	InsideTemp  float64 `json:"inside_temp"`
	OutsideTemp float64 `json:"outside_temp"`
	DriverTempSetting    float64 `json:"driver_temp_setting"`
	PassengerTempSetting float64 `json:"passenger_temp_setting"`
	IsACOn       bool  `json:"is_ac_on"`
	IsClimateOn  bool  `json:"is_climate_on"`
	Version      string `json:"version"`
	TpmsFL float64 `json:"tpms_fl"`
	TpmsFR float64 `json:"tpms_fr"`
	TpmsRL float64 `json:"tpms_rl"`
	TpmsRR float64 `json:"tpms_rr"`
	BatteryTemp    float64 `json:"battery_temp"`
	ChargerPhases  int     `json:"charger_phases"`
	Lightweight    bool    `json:"lightweight"`
	MediaPlaybackStatus string `json:"media_playback_status"`
	MediaAudioSource    string `json:"media_audio_source"`
	MediaVolume         int    `json:"media_volume"`
	NowPlayingTitle     string `json:"now_playing_title"`
	NowPlayingArtist    string `json:"now_playing_artist"`
	NowPlayingAlbum     string `json:"now_playing_album"`
	CenterDisplayState  int    `json:"center_display_state"`
	StateOutput    *state.VehicleStateOutput `json:"state_output,omitempty"`
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
				data.Response.MediaInfo.AudioSource)
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
		WindowsOpen: data.Response.VehicleState.WindowFL || data.Response.VehicleState.WindowFR || data.Response.VehicleState.WindowRL || data.Response.VehicleState.WindowRR,
		DoorOpen:    data.Response.VehicleState.DoorFL || data.Response.VehicleState.DoorFR || data.Response.VehicleState.DoorRL || data.Response.VehicleState.DoorRR,
		TrunkOpen:   data.Response.VehicleState.TrunkOpen,
		FrunkOpen:   data.Response.VehicleState.FrunkOpen,
		ChargePortOpen: data.Response.ChargeState.ChargePortOpen,
		DoorFL: data.Response.VehicleState.DoorFL,
		DoorFR: data.Response.VehicleState.DoorFR,
		DoorRL: data.Response.VehicleState.DoorRL,
		DoorRR: data.Response.VehicleState.DoorRR,
		WindowFL: data.Response.VehicleState.WindowFL,
		WindowFR: data.Response.VehicleState.WindowFR,
		WindowRL: data.Response.VehicleState.WindowRL,
		WindowRR: data.Response.VehicleState.WindowRR,
		InsideTemp:           data.Response.ClimateState.InsideTemp,
		OutsideTemp:          data.Response.ClimateState.OutsideTemp,
		DriverTempSetting:    data.Response.ClimateState.DriverTempSetting,
		PassengerTempSetting: data.Response.ClimateState.PassengerTempSetting,
		IsACOn:      data.Response.ClimateState.IsClimateOn,
		IsClimateOn: data.Response.ClimateState.IsClimateOn,
		Version:     data.Response.VehicleState.CarVersion,
		TpmsFL: data.Response.VehicleState.TPMSPressureFL,
		TpmsFR: data.Response.VehicleState.TPMSPressureFR,
		TpmsRL: data.Response.VehicleState.TPMSPressureRL,
		TpmsRR: data.Response.VehicleState.TPMSPressureRR,
		BatteryTemp:   data.Response.ClimateState.BatteryTemp,
		ChargerPhases: 0,
		Lightweight:   false,
		MediaPlaybackStatus: data.Response.MediaInfo.PlaybackStatus,
		MediaAudioSource:    data.Response.MediaInfo.AudioSource,
		MediaVolume:         data.Response.MediaInfo.Volume,
		NowPlayingTitle:     data.Response.MediaInfo.NowPlayingTitle,
		NowPlayingArtist:    data.Response.MediaInfo.NowPlayingArtist,
		NowPlayingAlbum:     data.Response.MediaInfo.NowPlayingAlbum,
		CenterDisplayState:  data.Response.VehicleState.CenterDisplayState,
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
	Hostname string                     `json:"hostname"`
	Fields   map[string]TelemetryField  `json:"fields"`
}

type TelemetryField struct {
	MinInterval int `json:"min_interval,omitempty"`
	MaxInterval int `json:"max_interval,omitempty"`
}

type TelemetryConfigResponse struct {
	Response struct {
		SuccessfulVINs  []string `json:"successful_vins"`
		SkippedVINs     []string `json:"skipped_vins,omitempty"`
	} `json:"response"`
	Error string `json:"error,omitempty"`
}

func ConfigureFleetTelemetry(accessToken string, vins []string, hostname string) (*TelemetryConfigResponse, error) {
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles/fleet_telemetry_config", cfg.Tesla.FleetAPIURL)

	mediaFields := map[string]TelemetryField{
		"MediaPlaybackStatus": {MinInterval: 1, MaxInterval: 600},
		"MediaAudioSource":    {MinInterval: 1, MaxInterval: 600},
		"MediaVolume":         {MinInterval: 1, MaxInterval: 600},
		"NowPlayingTitle":     {MinInterval: 1, MaxInterval: 600},
		"NowPlayingArtist":    {MinInterval: 1, MaxInterval: 600},
		"NowPlayingAlbum":     {MinInterval: 1, MaxInterval: 600},
		"NowPlayingDuration":  {MinInterval: 1, MaxInterval: 600},
		"NowPlayingElapsed":   {MinInterval: 1, MaxInterval: 600},
	}

	reqBody := TelemetryConfigRequest{
		VINs: vins,
		Config: TelemetryConfigPayload{
			Hostname: hostname,
			Fields:   mediaFields,
		},
	}

	log.Printf("[Fleet API] Configuring Fleet Telemetry for %d vehicles, hostname=%s", len(vins), hostname)

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("fleet telemetry config request failed: %w", err)
	}

	log.Printf("[Fleet API] fleet_telemetry_config status: %d", resp.StatusCode())

	var result TelemetryConfigResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse telemetry config response: %w", err)
	}

	if result.Error != "" {
		return &result, fmt.Errorf("fleet telemetry config error: %s", result.Error)
	}

	log.Printf("[Fleet API] Telemetry config: successful=%v, skipped=%v",
		result.Response.SuccessfulVINs, result.Response.SkippedVINs)

	return &result, nil
}
