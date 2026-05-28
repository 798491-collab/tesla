package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"tesla-server/config"
	"tesla-server/internal/ai"
	"tesla-server/internal/charging"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/middleware"
	"tesla-server/internal/polling"
	"tesla-server/internal/redis"
	"tesla-server/internal/tesla"
	"tesla-server/internal/trip"
	"tesla-server/internal/user"
	"tesla-server/internal/vcp"
	"tesla-server/internal/vehicle"
	"tesla-server/internal/ws"
	"tesla-server/models"
	"time"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	r.Use(middleware.CORS())

	api := r.Group("/api")
	{
		api.POST("/register", middleware.RateLimitAuth(), user.Register)
		api.POST("/login", middleware.RateLimitAuth(), user.Login)
		api.POST("/refresh-token", user.RefreshToken)
		api.GET("/tesla/auth", tesla.GetAuthURL)
		api.GET("/tesla/callback", tesla.Callback)
		api.GET("/tesla/auth_data", tesla.GetAuthData)
		api.POST("/tesla/partner/register", tesla.RegisterPartnerAccount)
		api.GET("/tesla/partner/check-public-key", tesla.CheckPartnerPublicKey)
		api.GET("/tesla/partner/check-hosting", tesla.CheckPublicKeyHosting)

		api.GET("/ws", ws.HandleWebSocket)
		api.GET("/ws/vin/:vin", ws.HandleWebSocketVIN)

		authorized := api.Group("")
		authorized.Use(middleware.JWTAuth())
		{
			authorized.POST("/logout", user.Logout)
			authorized.GET("/user/info", user.GetUserInfo)
			authorized.POST("/user/change_password", user.ChangePassword)
			authorized.POST("/user/update", user.UpdateUserInfo)

			authorized.POST("/tesla/bind", handleBindVehicle)
			authorized.GET("/tesla/vehicles", tesla.GetUserVehicles)
			authorized.GET("/tesla/vehicle/:vin/detail", tesla.GetVehicleDetail)
			authorized.GET("/tesla/vehicle/:vin/fleet-status", tesla.GetFleetStatus)
			authorized.GET("/tesla/vehicle/:vin/pairing-url", tesla.GetVirtualKeyPairingURL)
			authorized.DELETE("/tesla/unbind/:vin", handleUnbindVehicle)
			authorized.POST("/tesla/refresh-vehicle-info", tesla.RefreshVehicleInfo)

			authorized.GET("/vehicle/:vin/state", checkVehicleOwner, getVehicleState)
			authorized.POST("/vehicle/:vin/refresh", checkVehicleOwner, refreshVehicleState)
			authorized.POST("/vehicle/:vin/wake", checkVehicleOwner, wakeVehicle)
			authorized.GET("/vehicle/:vin/data", checkVehicleOwner, getVehicleData)

			authorized.GET("/trip/:vin/logs", checkVehicleOwner, getTripLogs)
			authorized.GET("/trip/:vin/stats", checkVehicleOwner, getTripStats)
			authorized.GET("/trip/:vin/monthly-list", checkVehicleOwner, getMonthlyTripList)
			authorized.GET("/trip/:vin/monthly-stats", checkVehicleOwner, getMonthlyTripStats)
			authorized.GET("/trip/:vin/points/:tripId", checkVehicleOwner, getTripPointsByVIN)
			authorized.GET("/vehicle/:vin/tracks", checkVehicleOwner, getVehicleTracks)

			authorized.GET("/charging/:vin/logs", checkVehicleOwner, getChargingLogs)
			authorized.GET("/charging/:vin/stats", checkVehicleOwner, getChargingStats)
			authorized.GET("/charging/:vin/monthly-list", checkVehicleOwner, getMonthlyChargingList)
			authorized.GET("/charging/:vin/monthly-stats", checkVehicleOwner, getMonthlyChargingStats)
			authorized.POST("/charging/log/:id/price", checkChargingLogOwner, updateChargingPrice)

			authorized.GET("/ai/trip/:vin/:refId", checkVehicleOwner, ai.GetTripAnalysis)
			authorized.GET("/ai/charging/:vin/:refId", checkVehicleOwner, ai.GetChargingAnalysis)
			authorized.GET("/ai/vehicle/:vin", checkVehicleOwner, ai.GetVehicleAnalysis)
			authorized.POST("/ai/trip/:vin/:refId", checkVehicleOwner, ai.TriggerTripAnalysis)
			authorized.POST("/ai/charging/:vin/:refId", checkVehicleOwner, ai.TriggerChargingAnalysis)
			authorized.POST("/ai/vehicle/:vin", checkVehicleOwner, ai.TriggerVehicleAnalysis)
			authorized.GET("/ai/history/:vin", checkVehicleOwner, ai.GetAnalysisHistory)
			authorized.GET("/ai/list/:vin", checkVehicleOwner, ai.GetAnalysisList)
			authorized.GET("/ai/latest/:vin/:type", checkVehicleOwner, ai.GetLatestAnalysis)

			authorized.POST("/vcp/door_lock", vcp.DoorLock)
			authorized.POST("/vcp/door_unlock", vcp.DoorUnlock)
			authorized.POST("/vcp/auto_conditioning_start", vcp.AutoConditioningStart)
			authorized.POST("/vcp/auto_conditioning_stop", vcp.AutoConditioningStop)
			authorized.POST("/vcp/honk_horn", vcp.HonkHorn)
			authorized.POST("/vcp/flash_lights", vcp.FlashLights)
			authorized.POST("/vcp/actuate_trunk", vcp.ActuateTrunk)
			authorized.POST("/vcp/actuate_frunk", vcp.ActuateFrunk)
			authorized.POST("/vcp/set_sentry_mode", vcp.SetSentryMode)
			authorized.POST("/vcp/charge_start", vcp.ChargeStart)
			authorized.POST("/vcp/charge_stop", vcp.ChargeStop)
			authorized.POST("/vcp/set_charge_limit", vcp.SetChargeLimit)
			authorized.POST("/vcp/charge_port_door_open", vcp.ChargePortDoorOpen)
			authorized.POST("/vcp/charge_port_door_close", vcp.ChargePortDoorClose)
			authorized.POST("/vcp/set_temps", vcp.SetTemps)
			authorized.POST("/vcp/remote_seat_heater", vcp.RemoteSeatHeater)
			authorized.POST("/vcp/remote_steering_wheel_heater", vcp.RemoteSteeringWheelHeater)
			authorized.GET("/vcp/commands", vcp.GetCommands)
		}
	}
}

func checkVehicleOwner(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)

	log.Printf("[checkVehicleOwner] Checking ownership: vin=%s, userID=%d", vin, userID)

	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND user_id = ? AND bind_status = 1", vin, userID).First(&vehicle).Error; err != nil {
		log.Printf("[checkVehicleOwner] Vehicle not found: vin=%s, userID=%d, error=%v", vin, userID, err)
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "No permission to access this vehicle"})
		c.Abort()
		return
	}

	log.Printf("[checkVehicleOwner] Found vehicle: vin=%s, userID=%d, bindStatus=%d", vin, userID, vehicle.BindStatus)
	c.Set("vehicleAccount", vehicle)
	c.Next()
}

func getVehicleState(c *gin.Context) {
	vin := c.Param("vin")

	data, err := polling.GetVehicleState(vin)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
		return
	}

	var v models.TeslaVehicle
	onlineState := "unknown"
	if err := database.DB.Where("vin = ?", vin).First(&v).Error; err == nil {
		if v.OnlineState != "" {
			onlineState = v.OnlineState
		}
	}

	log.Printf("[getVehicleState] No Redis cache for %s, fallback to DB online_state=%s", vin, onlineState)
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
		"vin":    vin,
		"online": onlineState != "offline",
		"state":  onlineState,
		"state_output": map[string]interface{}{
			"vin": vin,
			"state": map[string]interface{}{
				"online_state": onlineState,
				"confidence":   0.3,
				"changed_at":   time.Now().Unix(),
			},
			"drive": map[string]interface{}{
				"drive_state": "parked",
				"speed":       0,
				"gear":        "P",
			},
			"charge": map[string]interface{}{
				"charge_state": "disconnected",
				"battery_level": 0,
			},
			"lock": map[string]interface{}{
				"lock_state": "locked",
				"doors_open": false,
			},
			"command": map[string]interface{}{
				"command_state": "idle",
				"last_command":  "",
				"latency_ms":    0,
			},
		},
	}})
}

func refreshVehicleState(c *gin.Context) {
	vin := c.Param("vin")

	// 强制从 Tesla API 刷新
	data, err := polling.RefreshVehicleState(vin)
	if err != nil {
		errMsg := err.Error()
		// 车辆未找到
		if strings.Contains(errMsg, "vehicle not found") {
			c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
				"vin":    vin,
				"online": false,
				"state":  "offline",
			}})
			return
		}
		// Token 失效，需要重新授权
		if strings.Contains(errMsg, "token has been marked as invalid") ||
			strings.Contains(errMsg, "token refresh failed") {
			c.JSON(http.StatusOK, gin.H{
				"code":    401,
				"message": "Tesla 授权已过期，请重新绑定车辆",
				"data": gin.H{
					"vin":    vin,
					"online": false,
					"state":  "offline",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

func wakeVehicle(c *gin.Context) {
	vin := c.Param("vin")

	if err := polling.WakeVehicle(vin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "Vehicle wake up command sent"})
}

// getVehicleData 从 Tesla API 获取原始 vehicle_data
// 包含完整的车辆数据：位置、速度、电量等
// 如果 Tesla API 失败，返回缓存的最后状态（Last Known Good State）
func getVehicleData(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)

	// 获取车辆信息
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND vin = ? AND bind_status = 1", userID, vin).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "vehicle not found"})
		return
	}

	// 获取有效 token
	accessToken, err := tesla.GetValidAccessToken(vin)
	if err != nil {
		// Token 失效，返回缓存数据
		returnCachedState(c, vin, "token_invalid", err.Error())
		return
	}

	// 调用 Tesla vehicle_data API
	cfg := config.Load()
	url := fmt.Sprintf("%s/api/1/vehicles/%s/vehicle_data", cfg.Tesla.FleetAPIURL, vehicle.VehicleTag)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		returnCachedState(c, vin, "request_error", err.Error())
		return
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		returnCachedState(c, vin, "api_timeout", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		returnCachedState(c, vin, "read_error", err.Error())
		return
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		// Tesla API 返回错误（如 408 车辆睡眠/离线）
		returnCachedStateWithBody(c, vin, resp.StatusCode, body)
		return
	}

	// 检查位置权限
	var vehicleData map[string]interface{}
	if err := json.Unmarshal(body, &vehicleData); err == nil {
		if response, ok := vehicleData["response"].(map[string]interface{}); ok {
			if driveState, ok := response["drive_state"].(map[string]interface{}); ok {
				latitude, _ := driveState["latitude"].(float64)
				longitude, _ := driveState["longitude"].(float64)
				locationAuthorized := latitude != 0 && longitude != 0

				// 更新数据库
				database.DB.Model(&vehicle).Update("location_authorized", locationAuthorized)
			}
		}
	}

	c.Data(resp.StatusCode, "application/json", body)
}

// returnCachedState 返回缓存的最后状态
func returnCachedState(c *gin.Context, vin, errorType, errorMsg string) {
	log.Printf("[getVehicleData] Tesla API failed for %s: %s, returning cached state", vin, errorType)

	// 尝试从 Redis 获取缓存状态
	var cachedState fleet.SimpleVehicleData
	if err := redis.GetVehicleState(vin, &cachedState); err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"response":          cachedState,
				"state":             cachedState.State,
				"cached":            true,
				"last_success_at":   time.Now().Add(-5 * time.Minute).Unix(), // 估算时间
				"error_type":        errorType,
				"error_message":     errorMsg,
			},
		})
		return
	}

	// Redis 未命中，返回离线状态
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"response": gin.H{
				"vin":    vin,
				"online": false,
				"state":  "offline",
			},
			"state":         "offline",
			"cached":        false,
			"error_type":    errorType,
			"error_message": errorMsg,
		},
	})
}

// returnCachedStateWithBody 根据 Tesla 错误响应返回缓存状态
func returnCachedStateWithBody(c *gin.Context, vin string, statusCode int, body []byte) {
	log.Printf("[getVehicleData] Tesla API returned %d for %s, returning cached state", statusCode, vin)

	// 尝试解析 Tesla 的错误信息
	var errorResp map[string]interface{}
	errorMsg := "vehicle unavailable"
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if msg, ok := errorResp["error"].(string); ok {
			errorMsg = msg
		}
	}

	// 尝试从 Redis 获取缓存状态
	var cachedState fleet.SimpleVehicleData
	if err := redis.GetVehicleState(vin, &cachedState); err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"response":          cachedState,
				"state":             cachedState.State,
				"cached":            true,
				"last_success_at":   time.Now().Add(-5 * time.Minute).Unix(),
				"error_type":        "tesla_api_error",
				"error_message":     errorMsg,
				"tesla_status_code": statusCode,
			},
		})
		return
	}

	// Redis 未命中，返回离线状态
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"response": gin.H{
				"vin":    vin,
				"online": false,
				"state":  "offline",
			},
			"state":             "offline",
			"cached":            false,
			"error_type":        "tesla_api_error",
			"error_message":     errorMsg,
			"tesla_status_code": statusCode,
		},
	})
}

func getTripLogs(c *gin.Context) {
	vin := c.Param("vin")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var startDate, endDate time.Time
	if s := c.Query("start"); s != "" {
		startDate, _ = time.Parse("2006-01-02", s)
	}
	if e := c.Query("end"); e != "" {
		endDate, _ = time.Parse("2006-01-02", e)
	}

	logs, err := trip.GetTripLogs(vin, limit, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": logs})
}

func getTripPointsByVIN(c *gin.Context) {
	tripID, err := strconv.ParseUint(c.Param("tripId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid trip id"})
		return
	}

	points, err := trip.GetTripPoints(tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": points})
}

func getTripStats(c *gin.Context) {
	vin := c.Param("vin")
	startStr := c.Query("start")
	endStr := c.Query("end")

	startDate, _ := time.Parse("2006-01-02", startStr)
	endDate, _ := time.Parse("2006-01-02", endStr)
	if endDate.IsZero() {
		endDate = time.Now()
	}
	if startDate.IsZero() {
		startDate = endDate.AddDate(0, -1, 0)
	}

	stats, err := trip.GetTripStats(vin, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": stats})
}

// checkChargingLogOwner 检查用户是否有权限操作该充电记录
func checkChargingLogOwner(c *gin.Context) {
	logID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid log id"})
		c.Abort()
		return
	}

	userID := middleware.GetUserID(c)

	// 查询充电记录所属车辆
	var log models.ChargingLog
	if err := database.DB.First(&log, logID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "charging log not found"})
		c.Abort()
		return
	}

	// 检查车辆所有权
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND user_id = ? AND bind_status = 1", log.VIN, userID).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "no permission to access this charging log"})
		c.Abort()
		return
	}

	c.Set("chargingLog", log)
	c.Next()
}

// updateChargingPrice 更新充电记录价格
// 支持两种模式：
// 1. 慢充(AC)：传入 price_per_kwh（元/kWh），后端自动计算 total_cost
// 2. 快充(DC)：传入 total_cost（元），直接保存总金额
func updateChargingPrice(c *gin.Context) {
	logObj, exists := c.Get("chargingLog")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "charging log not found in context"})
		return
	}
	log := logObj.(models.ChargingLog)

	var req struct {
		PricePerKwh *float64 `json:"price_per_kwh"` // 慢充：电价（元/kWh）
		TotalCost   *float64 `json:"total_cost"`   // 快充：总费用（元）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request: " + err.Error()})
		return
	}

	// 验证至少传入一个字段
	if req.PricePerKwh == nil && req.TotalCost == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "price_per_kwh or total_cost is required"})
		return
	}

	updates := map[string]interface{}{}
	var pricePerKwh, totalCost float64

	if req.TotalCost != nil {
		// 快充模式：直接传入总金额
		totalCost = *req.TotalCost
		updates["total_cost"] = totalCost
		// 同时计算电价（方便统计平均电价）
		if log.ChargeKwh > 0 {
			pricePerKwh = totalCost / log.ChargeKwh
			updates["price_per_kwh"] = pricePerKwh
		}
	} else if req.PricePerKwh != nil {
		// 慢充模式：传入电价，计算总费用
		pricePerKwh = *req.PricePerKwh
		totalCost = pricePerKwh * log.ChargeKwh
		updates["price_per_kwh"] = pricePerKwh
		updates["total_cost"] = totalCost
	}

	// 更新数据库
	if err := database.DB.Model(&log).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to update price: " + err.Error()})
		return
	}

	// 返回更新后的记录
	log.PricePerKwh = &pricePerKwh
	log.TotalCost = &totalCost

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": log,
	})
}

func getMonthlyChargingStats(c *gin.Context) {
	vin := c.Param("vin")

	stats, err := charging.GetMonthlyChargingStats(vin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": stats})
}

func getMonthlyChargingList(c *gin.Context) {
	vin := c.Param("vin")

	list, err := charging.GetMonthlyChargingList(vin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list})
}

func getMonthlyTripStats(c *gin.Context) {
	vin := c.Param("vin")

	stats, err := trip.GetMonthlyStats(vin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": stats})
}

func getMonthlyTripList(c *gin.Context) {
	vin := c.Param("vin")

	list, err := trip.GetMonthlyTripList(vin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list})
}

func handleBindVehicle(c *gin.Context) {
	bodyBytes, _ := c.GetRawData()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	tesla.BindVehicle(c)

	// 如果 BindVehicle 已经返回了响应（成功或失败），不再继续执行
	if c.Writer.Written() {
		// 只有成功时才启动轮询
		if c.Writer.Status() == http.StatusOK {
			var req struct {
				VIN string `json:"vin"`
			}
			json.Unmarshal(bodyBytes, &req)
			if req.VIN != "" {
				// 刷新车辆映射缓存
				go func() {
					if err := vehicle.RefreshVehicleMapping(req.VIN); err != nil {
						log.Printf("[handleBindVehicle] Failed to refresh vehicle mapping: %v", err)
					}
				}()
				go polling.StartVehiclePolling(req.VIN)
			}
		}
		return
	}
}

func handleUnbindVehicle(c *gin.Context) {
	vin := c.Param("vin")
	tesla.UnbindVehicle(c)
	if c.Writer.Written() {
		return
	}
	// 清除缓存并停止轮询
	vehicle.DeleteVehicleMapping(vin)
	polling.StopVehiclePolling(vin)
	redis.DeleteVehicleState(vin)
}

func getChargingLogs(c *gin.Context) {
	vin := c.Param("vin")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var startDate, endDate time.Time
	if s := c.Query("start"); s != "" {
		startDate, _ = time.Parse("2006-01-02", s)
	}
	if e := c.Query("end"); e != "" {
		endDate, _ = time.Parse("2006-01-02", e)
	}

	logs, err := charging.GetChargingLogs(vin, limit, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": logs})
}

func getChargingStats(c *gin.Context) {
	vin := c.Param("vin")
	startStr := c.Query("start")
	endStr := c.Query("end")

	startDate, _ := time.Parse("2006-01-02", startStr)
	endDate, _ := time.Parse("2006-01-02", endStr)
	if endDate.IsZero() {
		endDate = time.Now()
	}
	if startDate.IsZero() {
		startDate = endDate.AddDate(0, -1, 0)
	}

	stats, err := charging.GetChargingStats(vin, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": stats})
}

func getVehicleTracks(c *gin.Context) {
	vin := c.Param("vin")

	startStr := c.Query("start")
	endStr := c.Query("end")

	var startTime, endTime time.Time
	if startStr != "" {
		startTime, _ = time.Parse("2006-01-02", startStr)
	}
	if endStr != "" {
		endTime, _ = time.Parse("2006-01-02", endStr)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}
	if startTime.IsZero() {
		startTime = endTime.AddDate(0, 0, -7)
	}

	points, err := trip.GetVehicleTrackPoints(vin, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	type trackPoint struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Speed     float64 `json:"speed"`
		Timestamp int64   `json:"timestamp"`
	}

	tracks := make([]trackPoint, 0, len(points))
	for _, p := range points {
		tracks = append(tracks, trackPoint{
			Latitude:  p.Latitude,
			Longitude: p.Longitude,
			Speed:     p.Speed,
			Timestamp: p.RecordedAt.UnixMilli(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": tracks})
}
