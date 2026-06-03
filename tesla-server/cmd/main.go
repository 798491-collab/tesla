package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"tesla-server/config"
	"tesla-server/internal/ai"
	"tesla-server/internal/auth"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
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
			keyData, err := os.ReadFile(cfg.Telemetry.PrivateKey)
			if err != nil {
				log.Printf("Warning: failed to read telemetry private key file %s: %v", cfg.Telemetry.PrivateKey, err)
			} else {
				privKeyPEM = keyData
			}
		}
		if err := telemetry.InitTelemetryServer(
			cfg.Telemetry.ListenAddr,
			privKeyPEM,
			cfg.Telemetry.TLSCertFile,
			cfg.Telemetry.TLSKeyFile,
			cfg.Telemetry.CACertFile,
			cfg.Telemetry.UseDefaultEngCA,
		); err != nil {
			log.Fatalf("Failed to start telemetry server: %v", err)
		}
		log.Println("Fleet Telemetry server started on", cfg.Telemetry.ListenAddr)

		// 延迟 10 秒后自动为已绑定车辆配置 Fleet Telemetry
		go func() {
			time.Sleep(10 * time.Second)

			if cfg.Telemetry.Hostname == "" {
				log.Println("[Telemetry Auto-Config] Skipped: TELEMETRY_HOSTNAME not configured")
				return
			}

			var vehicles []models.TeslaVehicle
			if err := database.DB.Where("bind_status = 1").Find(&vehicles).Error; err != nil {
				log.Printf("[Telemetry Auto-Config] Failed to query vehicles: %v", err)
				return
			}

			if len(vehicles) == 0 {
				log.Println("[Telemetry Auto-Config] No bound vehicles found")
				return
			}

			// 收集所有 VIN
			vins := make([]string, 0, len(vehicles))
			for _, v := range vehicles {
				vins = append(vins, v.VIN)
			}

			log.Printf("[Telemetry Auto-Config] Configuring Fleet Telemetry for %d vehicles: %v", len(vins), vins)

			// 获取任意一个车辆的 access token（这里简化处理，实际应该按用户分组）
			// 由于 ConfigureFleetTelemetry 需要 accessToken，我们需要从数据库获取
			// 简化方案：遍历每个车辆，获取其用户的 token 并配置

			configuredCount := 0
			for _, vehicle := range vehicles {
				// 获取该车辆所属用户的 token
				var account models.TeslaOAuthAccount
				if err := database.DB.Where("user_id = ? AND tesla_uid = ?", vehicle.UserID, vehicle.TeslaUID).First(&account).Error; err != nil {
					log.Printf("[Telemetry Auto-Config] Failed to get account for VIN %s: %v", vehicle.VIN, err)
					continue
				}

				if account.AccessToken == "" {
					log.Printf("[Telemetry Auto-Config] No access token for VIN %s", vehicle.VIN)
					continue
				}

				// 配置单个车辆，传入 CA 证书路径（使用 TLS 证书文件，包含证书链）
				resp, err := fleet.ConfigureFleetTelemetry(account.AccessToken, []string{vehicle.VIN}, cfg.Telemetry.Hostname, cfg.Telemetry.TLSCertFile)
				if err != nil {
					log.Printf("[Telemetry Auto-Config] Failed to configure VIN %s: %v", vehicle.VIN, err)
					continue
				}

				if len(resp.Response.SuccessfulVINs) > 0 || resp.Response.UpdatedVehicles > 0 {
					log.Printf("[Telemetry Auto-Config] Successfully configured VIN %s", vehicle.VIN)
					configuredCount++
				} else if len(resp.Response.SkippedVINs) > 0 {
					log.Printf("[Telemetry Auto-Config] VIN %s skipped (already configured or not supported)", vehicle.VIN)
				}

				// 避免请求过快
				time.Sleep(500 * time.Millisecond)
			}

			log.Printf("[Telemetry Auto-Config] Completed: %d/%d vehicles configured", configuredCount, len(vehicles))
		}()
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

	// 月度分析定时任务：每月1号凌晨0:10自动分析上月数据
	go func() {
		for {
			now := time.Now()
			// 下一个1号的0:10
			next := time.Date(now.Year(), now.Month()+1, 1, 0, 10, 0, 0, now.Location())
			if now.Day() == 1 && now.Hour() == 0 && now.Minute() < 15 {
				// 如果当前就是1号0点刚过，直接执行
				next = now
			}
			time.Sleep(next.Sub(now))

			log.Println("[AI Cron] Starting monthly analysis...")
			var vehicles []models.TeslaVehicle
			if err := database.DB.Where("bind_status = 1").Find(&vehicles).Error; err != nil {
				log.Printf("[AI Cron] Failed to query vehicles: %v", err)
				continue
			}

			lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")
			for _, v := range vehicles {
				// 月度行程分析
				tripRefID := fmt.Sprintf("trip_monthly:%s", lastMonth)
				go ai.RunTripAnalysis(v.VIN, v.UserID, tripRefID)
				time.Sleep(2 * time.Second)

				// 月度充电分析
				chargingRefID := fmt.Sprintf("charging_monthly:%s", lastMonth)
				go ai.RunChargingAnalysis(v.VIN, v.UserID, chargingRefID)
				time.Sleep(2 * time.Second)
			}
			log.Printf("[AI Cron] Monthly analysis triggered for %d vehicles (month: %s)", len(vehicles), lastMonth)
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
