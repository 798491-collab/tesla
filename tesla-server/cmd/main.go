package main

import (
	"log"
	"os"
	"path/filepath"
	"time"
	"tesla-server/config"
	"tesla-server/internal/ai"
	"tesla-server/internal/auth"
	"tesla-server/internal/database"
	"tesla-server/internal/logger"
	"tesla-server/internal/polling"
	"tesla-server/internal/redis"
	"tesla-server/internal/telemetry"
	"tesla-server/internal/ws"
	"tesla-server/models"
	"tesla-server/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	execPath, _ := os.Executable()
	logDir := filepath.Join(filepath.Dir(execPath), "log")
	if err := logger.Init(logDir); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()
	log.Println("Logger initialized, log directory:", logDir)

	cfg := config.Load()

	gin.SetMode(cfg.Server.Mode)

	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database connected successfully")

	if err := redis.Init(&cfg.Redis); err != nil {
		log.Fatalf("Failed to initialize redis: %v", err)
	}
	log.Println("Redis connected successfully")

	auth.Init(&cfg.JWT)

	polling.Init()
	log.Println("Vehicle polling service started")

	ws.InitHub()
	log.Println("WebSocket hub started")

	if cfg.Telemetry.Enabled {
		var privKeyPEM []byte
		if cfg.Telemetry.PrivateKey != "" {
			privKeyPEM = []byte(cfg.Telemetry.PrivateKey)
		}
		if err := telemetry.InitTelemetryServer(cfg.Telemetry.ListenAddr, privKeyPEM); err != nil {
			log.Fatalf("Failed to start telemetry server: %v", err)
		}
		log.Println("Fleet Telemetry server started on", cfg.Telemetry.ListenAddr)
	} else {
		log.Println("Fleet Telemetry server disabled (TELEMETRY_ENABLED not set)")
	}

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
			time.Sleep(next.Sub(now))
			logger.Rotate()
			log.Println("Log file rotated")
		}
	}()

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
			if next.Before(now) {
				next = next.AddDate(0, 0, 1)
			}
			time.Sleep(next.Sub(now))
			log.Println("[AI Cron] Starting daily vehicle analysis...")
			var vehicles []models.TeslaVehicle
			if err := database.DB.Where("bind_status = 1").Find(&vehicles).Error; err != nil {
				log.Printf("[AI Cron] Failed to query vehicles: %v", err)
				continue
			}
			for _, v := range vehicles {
				today := time.Now().Format("2006-01-02")
				go ai.RunVehicleAnalysis(v.VIN, v.UserID, today)
				time.Sleep(3 * time.Second)
			}
			log.Printf("[AI Cron] Daily vehicle analysis triggered for %d vehicles", len(vehicles))
		}
	}()

	r := gin.Default()
	routes.Setup(r)

	addr := ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
