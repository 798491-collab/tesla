package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"tesla-server/config"
	"tesla-server/internal/ai"
	"tesla-server/internal/database"
	"tesla-server/models"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	cfg := config.Load()
	if cfg.AI.APIKey == "" {
		log.Fatal("AI_API_KEY not configured. Please set AI_API_KEY in .env file or environment variable")
	}

	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	log.Println("Database connected")

	var vehicles []models.TeslaVehicle
	if err := database.DB.Where("bind_status = 1").Find(&vehicles).Error; err != nil {
		log.Fatalf("Failed to query vehicles: %v", err)
	}

	if len(vehicles) == 0 {
		log.Println("No bound vehicles found")
		return
	}

	log.Printf("Found %d bound vehicles\n", len(vehicles))

	for _, v := range vehicles {
		log.Printf("\n=== Processing VIN: %s (UserID: %d) ===\n", v.VIN, v.UserID)
		processVehicle(v)
	}

	log.Println("\n=== All done! ===")
}

func processVehicle(v models.TeslaVehicle) {
	processTrips(v)
	processCharging(v)
	processMonthlyTrips(v)
	processMonthlyCharging(v)
}

func processTrips(v models.TeslaVehicle) {
	var trips []models.TripLog
	database.DB.Where("vin = ? AND end_time IS NOT NULL", v.VIN).Find(&trips)
	log.Printf("  Trips: %d total\n", len(trips))

	analyzed := 0
	skipped := 0
	for _, trip := range trips {
		refID := fmt.Sprintf("trip:%d", trip.ID)

		var existing models.AIAnalysis
		if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", v.VIN, "trip", refID).First(&existing).Error; err == nil {
			skipped++
			continue
		}

		log.Printf("  Analyzing trip %d (%s)...\n", trip.ID, trip.StartTime.Format("2006-01-02 15:04"))
		ai.RunTripAnalysis(v.VIN, v.UserID, refID)
		analyzed++

		time.Sleep(2 * time.Second)
	}

	log.Printf("  Trips: analyzed=%d, skipped=%d\n", analyzed, skipped)
}

func processCharging(v models.TeslaVehicle) {
	var charges []models.ChargingLog
	database.DB.Where("vin = ? AND end_time IS NOT NULL", v.VIN).Find(&charges)
	log.Printf("  Charging logs: %d total\n", len(charges))

	analyzed := 0
	skipped := 0
	for _, charge := range charges {
		refID := fmt.Sprintf("charging:%d", charge.ID)

		var existing models.AIAnalysis
		if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", v.VIN, "charging", refID).First(&existing).Error; err == nil {
			skipped++
			continue
		}

		log.Printf("  Analyzing charge %d (%s)...\n", charge.ID, charge.StartTime.Format("2006-01-02 15:04"))
		ai.RunChargingAnalysis(v.VIN, v.UserID, refID)
		analyzed++

		time.Sleep(2 * time.Second)
	}

	log.Printf("  Charging: analyzed=%d, skipped=%d\n", analyzed, skipped)
}

func processMonthlyTrips(v models.TeslaVehicle) {
	var trips []models.TripLog
	database.DB.Where("vin = ?", v.VIN).
		Select("vin, start_time").
		Find(&trips)

	monthSet := make(map[string]bool)
	for _, trip := range trips {
		monthKey := trip.StartTime.Format("2006-01")
		monthSet[monthKey] = true
	}

	log.Printf("  Monthly trips: %d months\n", len(monthSet))

	analyzed := 0
	for month := range monthSet {
		refID := fmt.Sprintf("trip_monthly:%s", month)

		var existing models.AIAnalysis
		if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", v.VIN, "trip", refID).First(&existing).Error; err == nil {
			continue
		}

		log.Printf("  Analyzing monthly trip %s...\n", month)
		ai.RunTripAnalysis(v.VIN, v.UserID, refID)
		analyzed++

		time.Sleep(2 * time.Second)
	}

	log.Printf("  Monthly trips: analyzed=%d\n", analyzed)
}

func processMonthlyCharging(v models.TeslaVehicle) {
	var charges []models.ChargingLog
	database.DB.Where("vin = ?", v.VIN).
		Select("vin, start_time").
		Find(&charges)

	monthSet := make(map[string]bool)
	for _, charge := range charges {
		monthKey := charge.StartTime.Format("2006-01")
		monthSet[monthKey] = true
	}

	log.Printf("  Monthly charging: %d months\n", len(monthSet))

	analyzed := 0
	for month := range monthSet {
		refID := fmt.Sprintf("charging_monthly:%s", month)

		var existing models.AIAnalysis
		if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", v.VIN, "charging", refID).First(&existing).Error; err == nil {
			continue
		}

		log.Printf("  Analyzing monthly charging %s...\n", month)
		ai.RunChargingAnalysis(v.VIN, v.UserID, refID)
		analyzed++

		time.Sleep(2 * time.Second)
	}

	log.Printf("  Monthly charging: analyzed=%d\n", analyzed)
}

func init() {
	os.Setenv("TZ", "Asia/Shanghai")
}
