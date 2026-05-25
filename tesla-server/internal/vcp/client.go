package vcp

import (
	"fmt"
	"log"
	"net/http"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/polling"
	"tesla-server/internal/redis"
	"tesla-server/internal/vehicle"
	"tesla-server/models"
	"time"

	"github.com/gin-gonic/gin"
)

type CommandRequest struct {
	VIN   string `json:"vin" binding:"required"`
	Token string `json:"token"`
}

// checkVirtualKey 检查虚拟钥匙是否已配对
// 所有 VCP 命令都需要虚拟钥匙配对，否则 Tesla 会返回 "public key not paired"
func checkVirtualKey(vin string) error {
	v, err := vehicle.GetVehicleByVIN(vin)
	if err != nil {
		return fmt.Errorf("vehicle not found: %s", vin)
	}

	if v.VirtualKeyStatus == 1 {
		return nil
	}

	account, err := vehicle.GetOAuthAccountByVehicle(v)
	if err != nil {
		log.Printf("[VCP] Failed to get oauth account for %s: %v", vin, err)
		return fmt.Errorf("virtual key not paired. Please pair virtual key first via Tesla App")
	}

	status, err := fleet.VerifyVirtualKey(account.AccessToken, []string{vin})
	if err != nil {
		log.Printf("[VCP] Failed to verify virtual key for %s: %v", vin, err)
		return fmt.Errorf("virtual key not paired. Please pair virtual key first via Tesla App")
	}

	if !status.KeyPaired {
		return fmt.Errorf("virtual key not paired. Please pair virtual key first via Tesla App")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"virtual_key_status":     1,
		"virtual_key_paired_at":  now,
		"virtual_key_last_check": now,
	}
	database.DB.Model(&models.TeslaVehicle{}).Where("vin = ?", vin).Updates(updates)

	return nil
}

// wakeVehicleIfNeeded 检查车辆状态，如离线则唤醒
// wake_up 限制：5 分钟最多 1 次（通过 Redis 锁控制）
func wakeVehicleIfNeeded(vin string) error {
	var online bool
	if err := redis.Get(fmt.Sprintf("tesla:vehicle:%s:online", vin), &online); err == nil && online {
		return nil
	}

	mapping, err := vehicle.GetVehicleMapping(vin)
	if err != nil {
		return fmt.Errorf("vehicle not found: %s", vin)
	}

	log.Printf("[VCP] Vehicle %s appears offline, sending wake command...", vin)

	if err := fleet.WakeUp(mapping.AccessToken, mapping.VehicleTag); err != nil {
		return fmt.Errorf("failed to wake vehicle: %v", err)
	}

	log.Printf("[VCP] Wake command sent, waiting for vehicle to come online...")
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		data, err := fleet.GetVehicleState(mapping.AccessToken, mapping.VehicleTag)
		if err == nil && data.Online {
			log.Printf("[VCP] Vehicle %s is now online", vin)
			return nil
		}
	}

	return fmt.Errorf("vehicle did not come online after wake command")
}

// sendCommand 发送控制命令（带唤醒检查和限流）
// command 限制：最低间隔 2 秒
func sendCommand(vin, command string, body interface{}) (*fleet.CommandResponse, error) {
	accessToken, err := vehicle.GetValidAccessToken(vin)
	if err != nil {
		return nil, err
	}

	if err := checkVirtualKey(vin); err != nil {
		return nil, err
	}

	if command != "honk_horn" && command != "flash_lights" {
		if err := wakeVehicleIfNeeded(vin); err != nil {
			log.Printf("[VCP] Wake failed for %s: %v, attempting command anyway", vin, err)
		}
	}

	logCommand(vin, command, true, "")

	polling.SignalActivity(vin)

	return fleet.SendCommand(accessToken, vin, command, body)
}

// logCommand 记录控制命令日志
func logCommand(vin, command string, success bool, errorMessage string) {
	// 异步记录，不阻塞主流程
	go func() {
		logEntry := models.VehicleCommandLog{
			VIN:          vin,
			Command:      command,
			Success:      success,
			ErrorMessage: errorMessage,
		}
		if err := database.DB.Create(&logEntry).Error; err != nil {
			log.Printf("[VCP] Failed to log command: %v", err)
		}
	}()
}

func DoorLock(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "door_lock", nil)
	if err != nil {
		logCommand(req.VIN, "door_lock", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

func DoorUnlock(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "door_unlock", nil)
	if err != nil {
		logCommand(req.VIN, "door_unlock", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

func AutoConditioningStart(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "auto_conditioning_start", nil)
	if err != nil {
		logCommand(req.VIN, "auto_conditioning_start", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

func AutoConditioningStop(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "auto_conditioning_stop", nil)
	if err != nil {
		logCommand(req.VIN, "auto_conditioning_stop", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

func HonkHorn(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "honk_horn", nil)
	if err != nil {
		logCommand(req.VIN, "honk_horn", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

func FlashLights(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "flash_lights", nil)
	if err != nil {
		logCommand(req.VIN, "flash_lights", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// ActuateTrunk 控制后备箱
// POST /api/1/vehicles/{id}/command/actuate_trunk
// Body: { "which_trunk": "rear" }
func ActuateTrunk(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	body := map[string]string{
		"which_trunk": "rear",
	}

	resp, err := sendCommand(req.VIN, "actuate_trunk", body)
	if err != nil {
		logCommand(req.VIN, "actuate_trunk", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// ActuateFrunk 控制前备箱
// POST /api/1/vehicles/{id}/command/actuate_trunk
// Body: { "which_trunk": "front" }
func ActuateFrunk(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	body := map[string]string{
		"which_trunk": "front",
	}

	resp, err := sendCommand(req.VIN, "actuate_trunk", body)
	if err != nil {
		logCommand(req.VIN, "actuate_frunk", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// SetSentryMode 设置哨兵模式
// POST /api/1/vehicles/{id}/command/set_sentry_mode
// Body: { "on": true }
func SetSentryMode(c *gin.Context) {
	var req struct {
		VIN string `json:"vin" binding:"required"`
		On  bool   `json:"on"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	body := map[string]bool{
		"on": req.On,
	}

	resp, err := sendCommand(req.VIN, "set_sentry_mode", body)
	if err != nil {
		logCommand(req.VIN, "set_sentry_mode", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// ChargeStart 开始充电
// POST /api/1/vehicles/{id}/command/charge_start
func ChargeStart(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "charge_start", nil)
	if err != nil {
		logCommand(req.VIN, "charge_start", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// ChargeStop 停止充电
// POST /api/1/vehicles/{id}/command/charge_stop
func ChargeStop(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "charge_stop", nil)
	if err != nil {
		logCommand(req.VIN, "charge_stop", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// SetChargeLimit 设置充电限制
// POST /api/1/vehicles/{id}/command/set_charge_limit
// Body: { "percent": 80 }
func SetChargeLimit(c *gin.Context) {
	var req struct {
		VIN    string `json:"vin" binding:"required"`
		Percent int   `json:"percent" binding:"required,min=50,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	body := map[string]int{
		"percent": req.Percent,
	}

	resp, err := sendCommand(req.VIN, "set_charge_limit", body)
	if err != nil {
		logCommand(req.VIN, "set_charge_limit", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// ChargePortDoorOpen 打开充电口
// POST /api/1/vehicles/{id}/command/charge_port_door_open
func ChargePortDoorOpen(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "charge_port_door_open", nil)
	if err != nil {
		logCommand(req.VIN, "charge_port_door_open", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// ChargePortDoorClose 关闭充电口
// POST /api/1/vehicles/{id}/command/charge_port_door_close
func ChargePortDoorClose(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "charge_port_door_close", nil)
	if err != nil {
		logCommand(req.VIN, "charge_port_door_close", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// SetTemps 设置温度
// POST /api/1/vehicles/{id}/command/set_temps
// Body: { "driver_temp": 22, "passenger_temp": 22 }
func SetTemps(c *gin.Context) {
	var req struct {
		VIN           string  `json:"vin" binding:"required"`
		DriverTemp    float64 `json:"driver_temp" binding:"required"`
		PassengerTemp float64 `json:"passenger_temp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	body := map[string]float64{
		"driver_temp":    req.DriverTemp,
		"passenger_temp": req.PassengerTemp,
	}

	resp, err := sendCommand(req.VIN, "set_temps", body)
	if err != nil {
		logCommand(req.VIN, "set_temps", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// RemoteSeatHeater 座椅加热
// POST /api/1/vehicles/{id}/command/remote_seat_heater_request
// Body: { "heater": 0, "level": 3 }
func RemoteSeatHeater(c *gin.Context) {
	var req struct {
		VIN    string `json:"vin" binding:"required"`
		Heater int   `json:"heater" binding:"required,min=0,max=5"`
		Level  int   `json:"level" binding:"required,min=0,max=3"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	body := map[string]int{
		"heater": req.Heater,
		"level":  req.Level,
	}

	resp, err := sendCommand(req.VIN, "remote_seat_heater_request", body)
	if err != nil {
		logCommand(req.VIN, "remote_seat_heater_request", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

// RemoteSteeringWheelHeater 方向盘加热
// POST /api/1/vehicles/{id}/command/remote_steering_wheel_heater_request
func RemoteSteeringWheelHeater(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	resp, err := sendCommand(req.VIN, "remote_steering_wheel_heater_request", nil)
	if err != nil {
		logCommand(req.VIN, "remote_steering_wheel_heater_request", false, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Response,
	})
}

func GetCommands(c *gin.Context) {
	commands := []gin.H{
		{"command": "door_lock", "name": "锁车", "endpoint": "/api/vcp/door_lock"},
		{"command": "door_unlock", "name": "解锁", "endpoint": "/api/vcp/door_unlock"},
		{"command": "auto_conditioning_start", "name": "开空调", "endpoint": "/api/vcp/auto_conditioning_start"},
		{"command": "auto_conditioning_stop", "name": "关空调", "endpoint": "/api/vcp/auto_conditioning_stop"},
		{"command": "honk_horn", "name": "鸣笛", "endpoint": "/api/vcp/honk_horn"},
		{"command": "flash_lights", "name": "闪灯", "endpoint": "/api/vcp/flash_lights"},
		{"command": "actuate_trunk", "name": "后备箱", "endpoint": "/api/vcp/actuate_trunk"},
		{"command": "actuate_frunk", "name": "前备箱", "endpoint": "/api/vcp/actuate_frunk"},
		{"command": "set_sentry_mode", "name": "哨兵模式", "endpoint": "/api/vcp/set_sentry_mode"},
		{"command": "charge_start", "name": "开始充电", "endpoint": "/api/vcp/charge_start"},
		{"command": "charge_stop", "name": "停止充电", "endpoint": "/api/vcp/charge_stop"},
		{"command": "set_charge_limit", "name": "设置充电限制", "endpoint": "/api/vcp/set_charge_limit"},
		{"command": "charge_port_door_open", "name": "打开充电口", "endpoint": "/api/vcp/charge_port_door_open"},
		{"command": "charge_port_door_close", "name": "关闭充电口", "endpoint": "/api/vcp/charge_port_door_close"},
		{"command": "set_temps", "name": "设置温度", "endpoint": "/api/vcp/set_temps"},
		{"command": "remote_seat_heater", "name": "座椅加热", "endpoint": "/api/vcp/remote_seat_heater"},
		{"command": "remote_steering_wheel_heater", "name": "方向盘加热", "endpoint": "/api/vcp/remote_steering_wheel_heater"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": commands,
	})
}
