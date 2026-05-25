package vehicle

import (
	"fmt"
	"log"
	"tesla-server/internal/database"
	"tesla-server/internal/redis"
	"tesla-server/internal/tesla"
	"tesla-server/models"
)

// GetVehicleTag 获取 VehicleTag（优先 Redis，其次 MySQL）
// 这是 Service 层的核心函数，所有 Fleet API 调用前必须通过这里获取 vehicleTag
func GetVehicleTag(vin string) (string, error) {
	// 1. 先查 Redis
	mapping, err := redis.GetVehicleMapping(vin)
	if err == nil && mapping != nil && mapping.VehicleTag != "" {
		return mapping.VehicleTag, nil
	}

	// 2. Redis 未命中，查 MySQL
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&vehicle).Error; err != nil {
		return "", fmt.Errorf("vehicle not found in database: %s", vin)
	}

	// 3. 缓存到 Redis（30分钟 TTL）
	if vehicle.VehicleTag != "" {
		go func() {
			// 获取 access_token
			accessToken, _ := tesla.GetValidAccessToken(vin)
			mapping := &redis.VehicleMapping{
				VIN:         vin,
				VehicleTag:  vehicle.VehicleTag,
				AccessToken: accessToken,
				UserID:      vehicle.UserID,
			}
			if err := redis.SetVehicleMapping(mapping); err != nil {
				log.Printf("[VehicleMapping] Failed to cache mapping for %s: %v", vin, err)
			}
		}()
		return vehicle.VehicleTag, nil
	}

	return "", fmt.Errorf("vehicle %s has no vehicle_tag", vin)
}

// GetVehicleMapping 获取完整的车辆映射（包含 access_token）
// 优先 Redis，其次 MySQL
func GetVehicleMapping(vin string) (*redis.VehicleMapping, error) {
	// 1. 先查 Redis
	mapping, err := redis.GetVehicleMapping(vin)
	if err == nil && mapping != nil && mapping.VehicleTag != "" {
		return mapping, nil
	}

	// 2. Redis 未命中，查 MySQL
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&vehicle).Error; err != nil {
		return nil, fmt.Errorf("vehicle not found in database: %s", vin)
	}

	// 获取 access_token
	accessToken, err := tesla.GetValidAccessToken(vin)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	// 3. 缓存到 Redis
	mapping = &redis.VehicleMapping{
		VIN:         vin,
		VehicleTag:  vehicle.VehicleTag,
		AccessToken: accessToken,
		UserID:      vehicle.UserID,
	}
	go func() {
		if err := redis.SetVehicleMapping(mapping); err != nil {
			log.Printf("[VehicleMapping] Failed to cache mapping for %s: %v", vin, err)
		}
	}()

	return mapping, nil
}

// RefreshVehicleMapping 刷新 Redis 中的车辆映射
func RefreshVehicleMapping(vin string) error {
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&vehicle).Error; err != nil {
		return fmt.Errorf("vehicle not found: %s", vin)
	}

	// 获取有效的 access_token
	accessToken, err := tesla.GetValidAccessToken(vin)
	if err != nil {
		return fmt.Errorf("failed to get access token: %v", err)
	}

	mapping := &redis.VehicleMapping{
		VIN:         vin,
		VehicleTag:  vehicle.VehicleTag,
		AccessToken: accessToken,
		UserID:      vehicle.UserID,
	}
	return redis.SetVehicleMapping(mapping)
}

// DeleteVehicleMapping 删除车辆映射缓存
func DeleteVehicleMapping(vin string) error {
	return redis.DeleteVehicleMapping(vin)
}

// GetValidAccessToken 获取有效的 access_token（自动刷新）
// 这是 vehicle 层的封装，直接调用 tesla 层的实现
func GetValidAccessToken(vin string) (string, error) {
	return tesla.GetValidAccessToken(vin)
}

// GetVehicleByVIN 通过 VIN 获取车辆信息
func GetVehicleByVIN(vin string) (*models.TeslaVehicle, error) {
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&vehicle).Error; err != nil {
		return nil, fmt.Errorf("vehicle not found: %s", vin)
	}
	return &vehicle, nil
}

// GetOAuthAccountByVehicle 通过车辆获取 OAuth 账户
func GetOAuthAccountByVehicle(vehicle *models.TeslaVehicle) (*models.TeslaOAuthAccount, error) {
	var account models.TeslaOAuthAccount
	if err := database.DB.Where("tesla_uid = ?", vehicle.TeslaUID).First(&account).Error; err != nil {
		return nil, fmt.Errorf("oauth account not found for vehicle: %s", vehicle.VIN)
	}
	return &account, nil
}
