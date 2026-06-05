package models

import (
	"time"
)

// TeslaOAuthAccount Tesla OAuth 账户表
// Token 是账户级的，不是车辆级的
// 一个 Tesla 账户可能有 5 台车，token 是共用的
// 所以必须拆分为账户表 + 车辆表
type TeslaOAuthAccount struct {
	ID                   uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID               uint64    `gorm:"index" json:"user_id"`
	TeslaUID             string    `gorm:"size:64;index" json:"tesla_uid"`
	AccessToken          string    `gorm:"type:text" json:"-"`
	RefreshToken         string    `gorm:"type:text" json:"-"`
	ExpiresAt            int64     `json:"expires_at"`
	GrantedScopes        string    `gorm:"size:500" json:"granted_scopes"`
	TokenInvalid         bool      `gorm:"default:false" json:"token_invalid"`
	LastTokenRefreshAt   *time.Time `json:"last_token_refresh_at"`
	LastTokenRefreshError string   `gorm:"size:500" json:"last_token_refresh_error"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (TeslaOAuthAccount) TableName() string { return "tesla_oauth_accounts" }

// TeslaVehicle 车辆表
// 只保存车辆信息，不保存 token
// Token 通过 TeslaUID 关联到 TeslaOAuthAccount
type TeslaVehicle struct {
	ID                    uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                uint64    `gorm:"index" json:"user_id"`
	TeslaUID              string    `gorm:"size:64;index" json:"tesla_uid"`
	VIN                   string    `gorm:"size:32;index" json:"vin"`
	VehicleTag            string    `gorm:"size:64;index" json:"vehicle_tag"`
	DisplayName           string    `gorm:"size:64" json:"display_name"`
	AccessType            string    `gorm:"size:20" json:"access_type"`
	BindStatus            int8      `gorm:"default:1" json:"bind_status"`
	OnlineState           string    `gorm:"size:20" json:"online_state"`
	VirtualKeyStatus      int       `gorm:"default:0" json:"virtual_key_status"`
	VirtualKeyPairedAt    *time.Time `json:"virtual_key_paired_at"`
	VirtualKeyLastCheck   *time.Time `json:"virtual_key_last_check"`
	LocationAuthorized    bool      `gorm:"default:false" json:"location_authorized"`
	FleetTelemetryVersion string    `gorm:"size:32" json:"fleet_telemetry_version"`
	DiscountedDeviceData  bool      `gorm:"default:false" json:"discounted_device_data"`
	ApiVersion            int       `json:"api_version"`
	OptionCodes           string    `gorm:"size:500" json:"option_codes"`
	VehicleImage          string    `gorm:"size:500" json:"vehicle_image"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (TeslaVehicle) TableName() string { return "tesla_vehicles" }

// VehicleStateCache 车辆状态缓存
// 重要：只存 Redis，不进 MySQL
// 原因：车辆状态变化极其频繁（速度、SOC、空调、经纬度），每秒几百次 UPDATE 会炸数据库
// Redis Key: tesla:vehicle:{vin}:state
// TTL: 5分钟
type VehicleStateCache struct {
	ID                   uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN                  string    `gorm:"size:32;index" json:"vin"`
	Online               bool      `json:"online"`
	State                string    `gorm:"size:20" json:"state"`
	BatteryLevel         int       `json:"battery_level"`
	BatteryRangeKm       float64   `json:"battery_range_km"`
	ChargingState        string    `gorm:"size:20" json:"charging_state"`
	ChargeRate           float64   `json:"charge_rate"`
	ChargerPower         float64   `json:"charger_power"`
	ChargerActualCurrent float64   `json:"charger_actual_current"`
	ChargerVoltage       int       `json:"charger_voltage"`
	ChargeEnergyAdded    float64   `json:"charge_energy_added"`
	Speed                float64   `json:"speed"`
	ShiftState           string    `gorm:"size:5" json:"shift_state"`
	Latitude             float64   `json:"latitude"`
	Longitude            float64   `json:"longitude"`
	Heading              int       `json:"heading"`
	OdometerKm           float64   `json:"odometer_km"`
	Locked               bool      `json:"locked"`
	InsideTemp           float64   `json:"inside_temp"`
	OutsideTemp          float64   `json:"outside_temp"`
	DriverTempSetting    float64   `json:"driver_temp_setting"`
	PassengerTempSetting float64   `json:"passenger_temp_setting"`
	IsACOn               bool      `json:"is_ac_on"`
	CarVersion           string    `gorm:"size:64" json:"car_version"`
	TirePressureFL       float64   `json:"tire_pressure_fl"`
	TirePressureFR       float64   `json:"tire_pressure_fr"`
	TirePressureRL       float64   `json:"tire_pressure_rl"`
	TirePressureRR       float64   `json:"tire_pressure_rr"`
	SentryMode           bool      `json:"sentry_mode"`
	DoorFL               bool      `json:"door_fl"`
	DoorFR               bool      `json:"door_fr"`
	DoorRL               bool      `json:"door_rl"`
	DoorRR               bool      `json:"door_rr"`
	TrunkOpen            bool      `json:"trunk_open"`
	FrunkOpen            bool      `json:"frunk_open"`
	FdWindow            bool      `json:"fd_window"`
	FpWindow            bool      `json:"fp_window"`
	RdWindow            bool      `json:"rd_window"`
	RpWindow            bool      `json:"rp_window"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (VehicleStateCache) TableName() string { return "tesla_vehicle_state_caches" }

// TripLog 行驶记录表
type TripLog struct {
	ID                uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN               string     `gorm:"size:32;index" json:"vin"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           *time.Time `json:"end_time"`
	StartOdometer     float64    `json:"start_odometer"`
	EndOdometer       float64    `json:"end_odometer"`
	Distance          float64    `json:"distance"`
	AvgSpeed          float64    `json:"avg_speed"`
	MaxSpeed          float64    `json:"max_speed"`
	EnergyUsed        float64    `json:"energy_used"`
	StartBatteryLevel int        `json:"start_battery_level"`
	EndBatteryLevel   int        `json:"end_battery_level"`
	DriveDuration     int        `json:"drive_duration"`  // 行驶时间（秒）
	IdleDuration      int        `json:"idle_duration"`   // 堵车/怠速时间（秒）
	StartLat          float64    `json:"start_lat"`
	StartLng          float64    `json:"start_lng"`
	EndLat            float64    `json:"end_lat"`
	EndLng            float64    `json:"end_lng"`
	StartAddress      string     `gorm:"size:255" json:"start_address"`
	EndAddress        string     `gorm:"size:255" json:"end_address"`
	StartCity         string     `gorm:"size:100" json:"start_city"`
	EndCity           string     `gorm:"size:100" json:"end_city"`
	AvgConsumption    float64    `json:"avg_consumption"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (TripLog) TableName() string { return "tesla_trip_logs" }

// ChargingLog 充电记录表
type ChargingLog struct {
	ID                    uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN                   string     `gorm:"size:32;index" json:"vin"`
	StartTime             time.Time  `json:"start_time"`
	EndTime               *time.Time `json:"end_time"`
	SocStart              int        `json:"soc_start"`
	SocEnd                int        `json:"soc_end"`
	StartRange            float64    `json:"start_range"`
	EndRange              float64    `json:"end_range"`
	ChargeKwh             float64    `json:"charge_kwh"`
	EnergyAddedKwh        float64    `json:"energy_added_kwh"`
	MaxPower              float64    `json:"max_power"`
	AveragePowerKw        float64    `json:"average_power_kw"`
	PeakPowerKw           float64    `json:"peak_power_kw"`
	ChargeDurationMinutes int        `json:"charge_duration_minutes"`
	ChargeType            string     `gorm:"size:20" json:"charge_type"`
	IsDcFastCharge        bool       `json:"is_dc_fast_charge"`
	OutsideTemp           float64    `json:"outside_temp"`   // 车外温度（充电效率分析用）
	BatteryTemp           float64    `json:"battery_temp"`   // 电池温度（充电效率分析用）
	ChargerPhases         int       `json:"charger_phases"`
	FastChargerPresent    bool      `json:"fast_charger_present"`
	Location              string     `gorm:"size:255" json:"location"`
	Address               string     `gorm:"size:255" json:"address"`
	City                  string     `gorm:"size:100" json:"city"`
	District              string     `gorm:"size:100" json:"district"`
	PoiName               string     `gorm:"size:255" json:"poi_name"`
	Latitude              float64    `json:"latitude"`
	Longitude             float64    `json:"longitude"`
	PricePerKwh           *float64   `json:"price_per_kwh"`  // 电价(元/kWh)
	TotalCost             *float64   `json:"total_cost"`     // 总费用(元)
	CreatedAt             time.Time  `json:"created_at"`
}

func (ChargingLog) TableName() string { return "tesla_charging_logs" }

// VehicleTelemetry 车辆遥测记录表（原 VehicleState，避免与 fleet.VehicleState 冲突）
// 只保存重要状态变更事件，不保存实时状态
type VehicleTelemetry struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN           string    `gorm:"size:32;index" json:"vin"`
	BatteryLevel  int       `json:"battery_level"`
	ChargingState string    `gorm:"size:20" json:"charging_state"`
	Speed         float64   `json:"speed"`
	ShiftState    string    `gorm:"size:5" json:"shift_state"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Odometer      float64   `json:"odometer"`
	Locked        bool      `json:"locked"`
	InsideTemp    float64   `json:"inside_temp"`
	OutsideTemp   float64   `json:"outside_temp"`
	IsACOn        bool      `json:"is_ac_on"`
	Online        bool      `json:"online"`
	RecordedAt    time.Time `json:"recorded_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (VehicleTelemetry) TableName() string { return "tesla_vehicle_telemetries" }

// TripPoint 行驶轨迹点表
type TripPoint struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TripID       uint64    `gorm:"index" json:"trip_id"`
	VIN          string    `gorm:"size:32;index" json:"vin"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Speed        float64   `json:"speed"`
	Heading      int       `json:"heading"`
	BatteryLevel int       `json:"battery_level"`
	RecordedAt   time.Time `gorm:"index" json:"recorded_at"`
}

func (TripPoint) TableName() string { return "tesla_trip_points" }

// GeoCache 地理编码缓存表
// 使用 GeoHash 替代 float 做唯一索引，避免 GPS 精度问题
type GeoCache struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	GeoHash   string    `gorm:"size:16;uniqueIndex" json:"geo_hash"` // GeoHash，精度7位，约 150m
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Address   string    `gorm:"size:255" json:"address"`
	City      string    `gorm:"size:100" json:"city"`
	District  string    `gorm:"size:100" json:"district"`
	PoiName   string    `gorm:"size:255" json:"poi_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GeoCache) TableName() string { return "tesla_geo_caches" }

// VehicleCommandLog 车辆控制命令日志表
// 记录所有控制命令，用于审计和故障排查
type VehicleCommandLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN          string    `gorm:"size:32;index" json:"vin"`
	UserID       uint64    `gorm:"index" json:"user_id"`
	Command      string    `gorm:"size:64" json:"command"`
	Success      bool      `json:"success"`
	ErrorMessage string    `gorm:"size:500" json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}

func (VehicleCommandLog) TableName() string { return "tesla_vehicle_command_logs" }
