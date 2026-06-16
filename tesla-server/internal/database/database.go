package database

import (
	"fmt"
	"tesla-server/config"
	"tesla-server/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	return autoMigrate()
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.UserToken{},
		&models.TeslaOAuthAccount{},
		&models.TeslaVehicle{},
		&models.VehicleStateCache{},
		&models.TripLog{},
		&models.ChargingLog{},
		&models.VehicleTelemetry{},
		&models.TripPoint{},
		&models.GeoCache{},
		&models.VehicleCommandLog{},
		&models.AIAnalysis{},
		&models.TelemetryRealtime{},
		&models.TelemetryState{},
		&models.TelemetryMedia{},
		&models.TelemetryRaw{},
	)
}

func GetDB() *gorm.DB {
	return DB
}
