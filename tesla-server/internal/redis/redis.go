package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"tesla-server/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client
var ctx = context.Background()

func Init(cfg *config.RedisConfig) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	_, err := Client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	return nil
}

func Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, expiration).Err()
}

func Get(key string, dest interface{}) error {
	data, err := Client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func Delete(key string) error {
	return Client.Del(ctx, key).Err()
}

func Exists(key string) (bool, error) {
	n, err := Client.Exists(ctx, key).Result()
	return n > 0, err
}

func SetVehicleState(vin string, state interface{}) error {
	key := fmt.Sprintf("tesla:vehicle:%s:state", vin)

	// 将新数据序列化为 map
	newData, err := json.Marshal(state)
	if err != nil {
		return err
	}
	var newMap map[string]interface{}
	if err := json.Unmarshal(newData, &newMap); err != nil {
		return err
	}

	// 读取已有数据
	var existing map[string]interface{}
	val, err := Client.Get(ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(val), &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}

	// 增量合并：新值覆盖旧值，保留旧值中存在但新值中不存在的字段
	for k, v := range newMap {
		existing[k] = v
	}

	merged, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, merged, 0).Err()
}

func GetVehicleState(vin string, dest interface{}) error {
	key := fmt.Sprintf("tesla:vehicle:%s:state", vin)
	return Get(key, dest)
}

func DeleteVehicleState(vin string) error {
	key := fmt.Sprintf("tesla:vehicle:%s:state", vin)
	return Delete(key)
}

func UpdateVehicleStateFields(vin string, fields map[string]interface{}) error {
	key := fmt.Sprintf("tesla:vehicle:%s:state", vin)

	// Get existing state or create new
	var state map[string]interface{}
	val, err := Client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key doesn't exist, create new
		state = make(map[string]interface{})
	} else if err != nil {
		return err
	} else {
		if err := json.Unmarshal([]byte(val), &state); err != nil {
			return err
		}
	}

	// Apply updates
	for k, v := range fields {
		state[k] = v
	}

	// Save back
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, 0).Err() // state 键永不过期，保留最后已知状态
}

func SetVehicleOnline(vin string, online bool) error {
	key := fmt.Sprintf("tesla:vehicle:%s:online", vin)
	return Set(key, online, 5*time.Minute)
}

func SetVehicleCharging(vin string, charging bool) error {
	key := fmt.Sprintf("tesla:vehicle:%s:charging", vin)
	return Set(key, charging, 5*time.Minute)
}

// VehicleMapping VIN 到 VehicleTag 的映射缓存
type VehicleMapping struct {
	VIN         string `json:"vin"`
	VehicleTag  string `json:"vehicle_tag"`
	AccessToken string `json:"access_token"`
	UserID      uint64 `json:"user_id"`
	ExpiresAt   int64  `json:"expires_at"` // token 过期时间戳
}

// SetVehicleMapping 缓存 VIN → VehicleTag 映射
// TTL: 30分钟
func SetVehicleMapping(mapping *VehicleMapping) error {
	key := fmt.Sprintf("tesla:vin_map:%s", mapping.VIN)
	return Set(key, mapping, 30*time.Minute)
}

// GetVehicleMapping 从 Redis 获取 VIN → VehicleTag 映射
func GetVehicleMapping(vin string) (*VehicleMapping, error) {
	key := fmt.Sprintf("tesla:vin_map:%s", vin)
	var mapping VehicleMapping
	if err := Get(key, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

// DeleteVehicleMapping 删除 VIN 映射缓存
func DeleteVehicleMapping(vin string) error {
	key := fmt.Sprintf("tesla:vin_map:%s", vin)
	return Delete(key)
}

// SetWakeLock 设置 wake_up 锁，防止 wake 风暴
// TTL: 300秒（5分钟）
func SetWakeLock(vin string) error {
	key := fmt.Sprintf("tesla:wake:%s", vin)
	return Set(key, true, 5*time.Minute)
}

// HasWakeLock 检查 wake_up 锁是否存在
func HasWakeLock(vin string) (bool, error) {
	key := fmt.Sprintf("tesla:wake:%s", vin)
	return Exists(key)
}

// SetCommandLock 设置命令锁，防止 command 超限
// TTL: 2秒
func SetCommandLock(vin string) error {
	key := fmt.Sprintf("tesla:cmd:%s", vin)
	return Set(key, true, 2*time.Second)
}

// HasCommandLock 检查命令锁是否存在
func HasCommandLock(vin string) (bool, error) {
	key := fmt.Sprintf("tesla:cmd:%s", vin)
	return Exists(key)
}

// SetNX 设置键值，仅当键不存在时才成功（分布式锁）
func SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return Client.SetNX(ctx, key, data, expiration).Result()
}

// AcquireTokenRefreshLock 获取 token 刷新锁
// 防止多个请求同时刷新同一个 token
func AcquireTokenRefreshLock(vin string) (bool, error) {
	key := fmt.Sprintf("tesla:token_refresh_lock:%s", vin)
	return SetNX(key, true, 30*time.Second)
}

// ReleaseTokenRefreshLock 释放 token 刷新锁
func ReleaseTokenRefreshLock(vin string) error {
	key := fmt.Sprintf("tesla:token_refresh_lock:%s", vin)
	return Delete(key)
}

// SetPollingState 设置轮询状态
func SetPollingState(vin string, state interface{}) error {
	key := fmt.Sprintf("tesla:polling:%s", vin)
	return Set(key, state, 10*time.Minute)
}

// GetPollingState 获取轮询状态
func GetPollingState(vin string, dest interface{}) error {
	key := fmt.Sprintf("tesla:polling:%s", vin)
	return Get(key, dest)
}

// DeletePollingState 删除轮询状态
func SetVehicleRealtime(vin string, data interface{}) error {
	key := fmt.Sprintf("tesla:vehicle:%s:realtime", vin)
	return Set(key, data, 0) // 永不过期，保留最后已知实时数据
}

// UpdateVehicleRealtimeFields 增量更新实时数据（只覆盖实际推送的字段，不覆盖未推送的字段）
func UpdateVehicleRealtimeFields(vin string, fields map[string]interface{}) error {
	key := fmt.Sprintf("tesla:vehicle:%s:realtime", vin)

	// 获取现有数据
	var state map[string]interface{}
	val, err := Client.Get(ctx, key).Result()
	if err == redis.Nil {
		state = make(map[string]interface{})
	} else if err != nil {
		return err
	} else {
		if err := json.Unmarshal([]byte(val), &state); err != nil {
			// 如果已有数据格式不对，重新创建
			state = make(map[string]interface{})
		}
	}

	// 增量合并
	for k, v := range fields {
		state[k] = v
	}
	state["updated_at"] = time.Now().UnixMilli()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, 0).Err()
}

func GetVehicleRealtime(vin string, dest interface{}) error {
	key := fmt.Sprintf("tesla:vehicle:%s:realtime", vin)
	return Get(key, dest)
}

func DeleteVehicleRealtime(vin string) error {
	key := fmt.Sprintf("tesla:vehicle:%s:realtime", vin)
	return Delete(key)
}

type VehicleStatus struct {
	Online bool   `json:"online"`
	Source string `json:"source"`
}

func SetVehicleStatus(vin string, status *VehicleStatus) error {
	key := fmt.Sprintf("tesla:vehicle:%s:status", vin)
	return Set(key, status, 5*time.Minute)
}

func GetVehicleStatus(vin string) (*VehicleStatus, error) {
	key := fmt.Sprintf("tesla:vehicle:%s:status", vin)
	var status VehicleStatus
	if err := Get(key, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func DeletePollingState(vin string) error {
	key := fmt.Sprintf("tesla:polling:%s", vin)
	return Delete(key)
}
