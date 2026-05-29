package vcp

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/polling"
	"tesla-server/internal/redis"
	"tesla-server/internal/state"
	"tesla-server/internal/vehicle"
	"tesla-server/internal/ws"
	"tesla-server/models"
	"time"

	"github.com/gin-gonic/gin"
)

type CommandRequest struct {
	VIN   string `json:"vin" binding:"required"`
	Token string `json:"token"`
}

type CommandCapability string

const (
	CommandReady       CommandCapability = "ready"
	CommandWakeNeeded  CommandCapability = "wake_needed"
	CommandUnavailable CommandCapability = "unavailable"
)

var wakeRequiredErrors = []string{
	"vehicle unavailable",
	"vehicle asleep",
	"could_not_wake_buses",
	"upstream vehicle disconnected",
	"vehicle is offline",
}

var vcpKeyErrors = []string{
	"public key not paired",
	"virtual key not paired",
	"vcp required",
	"signature verification failed",
	"invalid signature",
	"missing shared key",
}

func IsWakeRequired(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, e := range wakeRequiredErrors {
		if strings.Contains(msg, e) {
			return true
		}
	}
	return false
}

func IsVCPKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, e := range vcpKeyErrors {
		if strings.Contains(msg, e) {
			return true
		}
	}
	return false
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded")
}

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

func getCommandCapability(vin string) CommandCapability {
	var online bool
	if err := redis.Get(fmt.Sprintf("tesla:vehicle:%s:online", vin), &online); err == nil && online {
		return CommandReady
	}

	var pollingState string
	if err := redis.Get(fmt.Sprintf("tesla:polling:%s", vin), &pollingState); err == nil {
		if pollingState == "online" || pollingState == "driving" || pollingState == "charging" || pollingState == "climate_on" {
			return CommandReady
		}
	}

	lastSuccessKey := fmt.Sprintf("tesla:vehicle:%s:last_success", vin)
	var lastSuccess int64
	if err := redis.Get(lastSuccessKey, &lastSuccess); err == nil {
		if time.Since(time.Unix(lastSuccess, 0)) < 10*time.Minute {
			return CommandWakeNeeded
		}
	}

	return CommandUnavailable
}

func wakeVehicleAsync(vin string, command string, body interface{}) {
	log.Printf("[VCP] Starting async wake for %s after command '%s' failed", vin, command)

	mapping, err := vehicle.GetVehicleMapping(vin)
	if err != nil {
		log.Printf("[VCP] Wake failed for %s: mapping error: %v", vin, err)
		state.RecordCommandResult(vin, false)
		broadcastCommandState(vin, "failed", command, 0)
		return
	}

	if err := fleet.WakeUp(mapping.AccessToken, mapping.VehicleTag); err != nil {
		log.Printf("[VCP] Wake command failed for %s: %v", vin, err)
		state.RecordCommandResult(vin, false)
		broadcastCommandState(vin, "failed", command, 0)
		return
	}

	log.Printf("[VCP] Wake command sent for %s, waiting for vehicle to come online...", vin)
	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		data, err := fleet.GetVehicleState(mapping.AccessToken, mapping.VehicleTag)
		if err == nil && data.Online {
			log.Printf("[VCP] Vehicle %s is now online, retrying command '%s'", vin, command)
			accessToken, err := vehicle.GetValidAccessToken(vin)
			if err != nil {
				state.RecordCommandResult(vin, false)
				broadcastCommandState(vin, "failed", command, 0)
				return
			}
			resp, retryErr := fleet.SendCommand(accessToken, vin, command, body)
			if retryErr != nil {
				log.Printf("[VCP] Retry command '%s' failed for %s: %v", command, vin, retryErr)
				state.RecordCommandResult(vin, false)
				broadcastCommandState(vin, "failed", command, 0)
			} else {
				log.Printf("[VCP] Retry command '%s' succeeded for %s", command, vin)
				state.RecordCommandResult(vin, true)
				broadcastCommandState(vin, "success", command, 0)
				logCommand(vin, command, true, "retry_after_wake")
				_ = resp
			}
			polling.SignalActivity(vin)
			return
		}
	}

	log.Printf("[VCP] Vehicle %s did not come online after wake", vin)
	state.RecordCommandResult(vin, false)
	broadcastCommandState(vin, "failed", command, 0)
}

func broadcastCommandState(vin string, cmdState string, command string, latencyMs int64) {
	ws.BroadcastCommandState(vin, cmdState, command, latencyMs)
}

func sendCommand(vin, command string, body interface{}) (*fleet.CommandResponse, error) {
	accessToken, err := vehicle.GetValidAccessToken(vin)
	if err != nil {
		return nil, err
	}

	if err := checkVirtualKey(vin); err != nil {
		return nil, err
	}

	capability := getCommandCapability(vin)
	log.Printf("[VCP] Command '%s' for %s, capability: %s", command, vin, capability)

	state.RecordCommandStart(vin, command)
	broadcastCommandState(vin, "sending", command, 0)

	resp, err := fleet.SendCommand(accessToken, vin, command, body)

	if err == nil {
		state.RecordCommandResult(vin, true)
		broadcastCommandState(vin, "success", command, 0)
		logCommand(vin, command, true, "")

		polling.SignalActivity(vin)
		go func() {
			time.Sleep(3 * time.Second)
			polling.SignalActivity(vin)
		}()

		return resp, nil
	}

	if IsVCPKeyError(err) {
		state.RecordCommandResult(vin, false)
		broadcastCommandState(vin, "failed", command, 0)
		return nil, err
	}

	if IsTimeoutError(err) {
		log.Printf("[VCP] Command '%s' timeout for %s, treating as pending", command, vin)
		broadcastCommandState(vin, "pending", command, 0)

		go func() {
			time.Sleep(5 * time.Second)
			polling.SignalActivity(vin)
		}()

		return nil, &CommandPendingError{Command: command, VIN: vin}
	}

	if IsWakeRequired(err) {
		log.Printf("[VCP] Command '%s' for %s needs wake, starting async wake", command, vin)
		broadcastCommandState(vin, "waking", command, 0)

		go wakeVehicleAsync(vin, command, body)

		return nil, &CommandWakingError{Command: command, VIN: vin}
	}

	state.RecordCommandResult(vin, false)
	broadcastCommandState(vin, "failed", command, 0)
	return nil, err
}

type CommandPendingError struct {
	Command string
	VIN     string
}

func (e *CommandPendingError) Error() string {
	return fmt.Sprintf("command '%s' timeout, result pending", e.Command)
}

type CommandWakingError struct {
	Command string
	VIN     string
}

func (e *CommandWakingError) Error() string {
	return fmt.Sprintf("vehicle waking, command '%s' will retry automatically", e.Command)
}

func logCommand(vin, command string, success bool, errorMessage string) {
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

func handleCommandResponse(c *gin.Context, vin, command string, resp *fleet.CommandResponse, err error) {
	if err != nil {
		switch err.(type) {
		case *CommandWakingError:
			c.JSON(http.StatusAccepted, gin.H{
				"code":    202,
				"status":  "waking",
				"message": "车辆唤醒中，命令将自动重试",
				"command": command,
			})
		case *CommandPendingError:
			c.JSON(http.StatusAccepted, gin.H{
				"code":    202,
				"status":  "pending",
				"message": "命令已发送，等待车辆确认",
				"command": command,
			})
		default:
			errMsg := err.Error()
			if IsVCPKeyError(err) {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"status":  "vcp_required",
					"message": errMsg,
					"command": command,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"status":  "failed",
				"message": errMsg,
				"command": command,
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"status": "success",
		"data":   resp.Response,
	})
}

func DoorLock(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "door_lock", nil)
	handleCommandResponse(c, req.VIN, "door_lock", resp, err)
}

func DoorUnlock(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "door_unlock", nil)
	handleCommandResponse(c, req.VIN, "door_unlock", resp, err)
}

func AutoConditioningStart(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "auto_conditioning_start", nil)
	handleCommandResponse(c, req.VIN, "auto_conditioning_start", resp, err)
}

func AutoConditioningStop(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "auto_conditioning_stop", nil)
	handleCommandResponse(c, req.VIN, "auto_conditioning_stop", resp, err)
}

func HonkHorn(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "honk_horn", nil)
	handleCommandResponse(c, req.VIN, "honk_horn", resp, err)
}

func FlashLights(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "flash_lights", nil)
	handleCommandResponse(c, req.VIN, "flash_lights", resp, err)
}

func ActuateTrunk(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	body := map[string]string{"which_trunk": "rear"}
	resp, err := sendCommand(req.VIN, "actuate_trunk", body)
	handleCommandResponse(c, req.VIN, "actuate_trunk", resp, err)
}

func ActuateFrunk(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	body := map[string]string{"which_trunk": "front"}
	resp, err := sendCommand(req.VIN, "actuate_trunk", body)
	handleCommandResponse(c, req.VIN, "actuate_frunk", resp, err)
}

func SetSentryMode(c *gin.Context) {
	var req struct {
		VIN string `json:"vin" binding:"required"`
		On  bool   `json:"on"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	body := map[string]bool{"on": req.On}
	resp, err := sendCommand(req.VIN, "set_sentry_mode", body)
	handleCommandResponse(c, req.VIN, "set_sentry_mode", resp, err)
}

func ChargeStart(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "charge_start", nil)
	handleCommandResponse(c, req.VIN, "charge_start", resp, err)
}

func ChargeStop(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "charge_stop", nil)
	handleCommandResponse(c, req.VIN, "charge_stop", resp, err)
}

func SetChargeLimit(c *gin.Context) {
	var req struct {
		VIN     string `json:"vin" binding:"required"`
		Percent int    `json:"percent" binding:"required,min=50,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	body := map[string]int{"percent": req.Percent}
	resp, err := sendCommand(req.VIN, "set_charge_limit", body)
	handleCommandResponse(c, req.VIN, "set_charge_limit", resp, err)
}

func ChargePortDoorOpen(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "charge_port_door_open", nil)
	handleCommandResponse(c, req.VIN, "charge_port_door_open", resp, err)
}

func ChargePortDoorClose(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "charge_port_door_close", nil)
	handleCommandResponse(c, req.VIN, "charge_port_door_close", resp, err)
}

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
	body := map[string]float64{"driver_temp": req.DriverTemp, "passenger_temp": req.PassengerTemp}
	resp, err := sendCommand(req.VIN, "set_temps", body)
	handleCommandResponse(c, req.VIN, "set_temps", resp, err)
}

func RemoteSeatHeater(c *gin.Context) {
	var req struct {
		VIN    string `json:"vin" binding:"required"`
		Heater int    `json:"heater" binding:"required,min=0,max=5"`
		Level  int    `json:"level" binding:"required,min=0,max=3"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	body := map[string]int{"heater": req.Heater, "level": req.Level}
	resp, err := sendCommand(req.VIN, "remote_seat_heater_request", body)
	handleCommandResponse(c, req.VIN, "remote_seat_heater_request", resp, err)
}

func RemoteSteeringWheelHeater(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "remote_steering_wheel_heater_request", nil)
	handleCommandResponse(c, req.VIN, "remote_steering_wheel_heater_request", resp, err)
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
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": commands})
}

func MediaTogglePlayback(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_toggle_playback", nil)
	handleCommandResponse(c, req.VIN, "media_toggle_playback", resp, err)
}

func MediaNextTrack(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_next_track", nil)
	handleCommandResponse(c, req.VIN, "media_next_track", resp, err)
}

func MediaPrevTrack(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_prev_track", nil)
	handleCommandResponse(c, req.VIN, "media_prev_track", resp, err)
}

func MediaNextFav(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_next_fav", nil)
	handleCommandResponse(c, req.VIN, "media_next_fav", resp, err)
}

func MediaPrevFav(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_prev_fav", nil)
	handleCommandResponse(c, req.VIN, "media_prev_fav", resp, err)
}

func MediaVolumeUp(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_volume_up", nil)
	handleCommandResponse(c, req.VIN, "media_volume_up", resp, err)
}

func MediaVolumeDown(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := sendCommand(req.VIN, "media_volume_down", nil)
	handleCommandResponse(c, req.VIN, "media_volume_down", resp, err)
}

func AdjustVolume(c *gin.Context) {
	var req struct {
		VIN    string `json:"vin" binding:"required"`
		Volume int    `json:"volume" binding:"required,min=0,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	body := map[string]int{"volume": req.Volume}
	resp, err := sendCommand(req.VIN, "adjust_volume", body)
	handleCommandResponse(c, req.VIN, "adjust_volume", resp, err)
}
