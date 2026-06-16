package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONMap 自定义类型，用于 GORM 存储 map[string]interface{} 为 JSON
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	}
	return json.Unmarshal(bytes, j)
}

// TelemetryRealtime 遥测实时数据原始记录
// 高频数据：速度、档位、功率、位置、SOC 等
type TelemetryRealtime struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN       string    `gorm:"size:32;index" json:"vin"`
	Data      JSONMap   `gorm:"type:json" json:"data"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (TelemetryRealtime) TableName() string { return "telemetry_realtime" }

// TelemetryState 遥测车辆状态原始记录
// 低频数据：门锁、车窗、空调、胎压、充电详情、安全设置等
type TelemetryState struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN       string    `gorm:"size:32;index" json:"vin"`
	Data      JSONMap   `gorm:"type:json" json:"data"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (TelemetryState) TableName() string { return "telemetry_state" }

// TelemetryMedia 遥测媒体状态原始记录
// 媒体播放：播放状态、音源、曲目信息等
type TelemetryMedia struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN       string    `gorm:"size:32;index" json:"vin"`
	Data      JSONMap   `gorm:"type:json" json:"data"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (TelemetryMedia) TableName() string { return "telemetry_media" }

// TelemetryRaw 遥测原始二进制记录
// 存储车辆推送的原始 Flatbuffers 数据，用于事后分析和排查
type TelemetryRaw struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VIN       string    `gorm:"size:32;index" json:"vin"`
	Topic     string    `gorm:"size:32;index" json:"topic"`
	Txid      string    `gorm:"size:64" json:"txid"`
	RawData   []byte    `gorm:"type:mediumblob" json:"raw_data"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (TelemetryRaw) TableName() string { return "telemetry_raw" }
