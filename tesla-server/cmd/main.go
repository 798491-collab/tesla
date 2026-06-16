package main

import (
	"fmt"
	"io"
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

	// 证书同步：从宝塔证书目录和 VCP 密钥目录同步到 certs 目录
	if cfg.Telemetry.Enabled && cfg.Telemetry.CertSync.Enabled {
		syncCerts(cfg)
	}

	gin.SetMode(cfg.Server.Mode)

	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := redis.Init(&cfg.Redis); err != nil {
		log.Fatalf("Failed to initialize redis: %v", err)
	}

	auth.Init(&cfg.JWT)
	polling.Init()
	ws.InitHub()

	if cfg.Telemetry.Enabled {
		var privKeyPEM []byte
		if cfg.Telemetry.PrivateKey != "" {
			keyData, err := os.ReadFile(cfg.Telemetry.PrivateKey)
			if err != nil {
				log.Printf("[Startup] Private key file not found: %s (%v)", cfg.Telemetry.PrivateKey, err)
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
			log.Fatalf("[Startup] Fleet Telemetry server failed: %v", err)
		}

		// telemetry.InitWriter() // 已关闭数据库写入
		// defer telemetry.StopWriter()

		// 检查 TLS 证书
		if cfg.Telemetry.TLSCertFile != "" {
			if _, err := os.Stat(cfg.Telemetry.TLSCertFile); err != nil {
				log.Printf("[Startup] TLS cert not found: %s", cfg.Telemetry.TLSCertFile)
			}
		}
		if cfg.Telemetry.TLSKeyFile != "" {
			if _, err := os.Stat(cfg.Telemetry.TLSKeyFile); err != nil {
				log.Printf("[Startup] TLS key not found: %s", cfg.Telemetry.TLSKeyFile)
			}
		}

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

			vins := make([]string, 0, len(vehicles))
			for _, v := range vehicles {
				vins = append(vins, v.VIN)
			}

			configuredCount := 0
			for _, vehicle := range vehicles {
				var account models.TeslaOAuthAccount
				if err := database.DB.Where("user_id = ? AND tesla_uid = ?", vehicle.UserID, vehicle.TeslaUID).First(&account).Error; err != nil {
					log.Printf("[Telemetry Auto-Config] No account for VIN %s: %v", vehicle.VIN, err)
					continue
				}

				if account.AccessToken == "" {
					log.Printf("[Telemetry Auto-Config] No access token for VIN %s", vehicle.VIN)
					continue
				}

				resp, err := fleet.ConfigureFleetTelemetry(account.AccessToken, []string{vehicle.VIN}, cfg.Telemetry.Hostname, cfg.Telemetry.TLSCertFile)
				if err != nil {
					log.Printf("[Telemetry Auto-Config] Failed VIN %s: %v", vehicle.VIN, err)
					continue
				}

				if len(resp.Response.SuccessfulVINs) > 0 || resp.Response.UpdatedVehicles > 0 {
					configuredCount++
				}

				time.Sleep(500 * time.Millisecond)
			}

			log.Printf("[Telemetry Auto-Config] Completed: %d/%d vehicles", configuredCount, len(vehicles))
		}()
	} else {
		log.Println("[Startup] Fleet Telemetry disabled")
	}

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
			time.Sleep(next.Sub(now))
			logger.Rotate()
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
			var vehicles []models.TeslaVehicle
			if err := database.DB.Where("bind_status = 1").Find(&vehicles).Error; err != nil {
				continue
			}
			for _, v := range vehicles {
				today := time.Now().Format("2006-01-02")
				go ai.RunVehicleAnalysis(v.VIN, v.UserID, today)
				time.Sleep(3 * time.Second)
			}
		}
	}()

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month()+1, 1, 0, 10, 0, 0, now.Location())
			if now.Day() == 1 && now.Hour() == 0 && now.Minute() < 15 {
				next = now
			}
			time.Sleep(next.Sub(now))

			var vehicles []models.TeslaVehicle
			if err := database.DB.Where("bind_status = 1").Find(&vehicles).Error; err != nil {
				continue
			}

			lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")
			for _, v := range vehicles {
				tripRefID := fmt.Sprintf("trip_monthly:%s", lastMonth)
				go ai.RunTripAnalysis(v.VIN, v.UserID, tripRefID)
				time.Sleep(2 * time.Second)

				chargingRefID := fmt.Sprintf("charging_monthly:%s", lastMonth)
				go ai.RunChargingAnalysis(v.VIN, v.UserID, chargingRefID)
				time.Sleep(2 * time.Second)
			}
		}
	}()

	r := gin.Default()
	routes.Setup(r)

	addr := ":" + cfg.Server.Port
	log.Printf("[Startup] Server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// syncCerts 从宝塔证书目录和 VCP 密钥目录同步证书到 certs 目录
func syncCerts(cfg *config.Config) {
	cs := cfg.Telemetry.CertSync

	// 自动推导 certs 目录：与可执行文件同目录下的 certs/
	execPath, _ := os.Executable()
	appDir := filepath.Dir(execPath)
	certsDir := cs.CertsDir
	if certsDir == "" {
		certsDir = filepath.Join(appDir, "certs")
	}

	os.MkdirAll(certsDir, 0755)

	// 同步 VCP 私钥
	if cs.VCPKeysDir != "" {
		syncIfNewer(filepath.Join(cs.VCPKeysDir, "private.pem"), filepath.Join(certsDir, "private.pem"))
	}

	// 同步宝塔 TLS 证书
	if cs.CertSrcDir != "" {
		syncIfNewer(filepath.Join(cs.CertSrcDir, "fullchain.pem"), filepath.Join(certsDir, "fullchain.pem"))
		syncIfNewer(filepath.Join(cs.CertSrcDir, "privkey.pem"), filepath.Join(certsDir, "privkey.pem"))
	}
}

// syncIfNewer 仅在源文件比目标文件新时复制
func syncIfNewer(src, dst string) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		log.Printf("[cert-sync] Source not found: %s", src)
		return
	}

	dstInfo, dstErr := os.Stat(dst)
	if dstErr == nil && !srcInfo.ModTime().After(dstInfo.ModTime()) {
		return // 目标已存在且不旧，跳过
	}

	srcFile, err := os.Open(src)
	if err != nil {
		log.Printf("[cert-sync] Failed to open %s: %v", src, err)
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		log.Printf("[cert-sync] Failed to create %s: %v", dst, err)
		return
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		log.Printf("[cert-sync] Failed to copy %s -> %s: %v", src, dst, err)
		return
	}

	os.Chmod(dst, 0644)
	log.Printf("[cert-sync] Synced: %s -> %s", src, dst)
}
