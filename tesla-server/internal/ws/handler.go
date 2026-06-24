package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"tesla-server/internal/auth"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	redispkg "tesla-server/internal/redis"
	vstate "tesla-server/internal/state"
	"tesla-server/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Sec-WebSocket-Protocol")
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token required"})
		return
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
		return
	}

	userID := claims.UserID

	var vehicles []models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND bind_status = 1", userID).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "database error"})
		return
	}

	vins := make(map[string]bool)
	for _, v := range vehicles {
		vins[v.VIN] = true
	}

	if len(vins) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "no vehicles bound"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		VINs:   vins,
		Hub:    DefaultHub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	DefaultHub.Register(client)

	go client.WritePump()
	go client.ReadPump()

	log.Printf("[WS] New connection: user_id=%d, vins=%v", userID, vins)
}

func HandleWebSocketVIN(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Sec-WebSocket-Protocol")
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token required"})
		return
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
		return
	}

	vin := c.Param("vin")
	if vin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "vin required"})
		return
	}

	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND user_id = ? AND bind_status = 1", vin, claims.UserID).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "no permission"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	vins := map[string]bool{vin: true}

	client := &Client{
		UserID: claims.UserID,
		VINs:   vins,
		Hub:    DefaultHub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	DefaultHub.Register(client)

	go client.WritePump()
	go client.ReadPump()

	// 连接建立后立即推送 Redis 中的当前车辆状态，确保前端在睡眠/离线时也能显示最后已知数据
	go pushInitialVehicleState(client, vin)

	log.Printf("[WS] New VIN connection: user_id=%d, vin=%s", claims.UserID, vin)
}

// pushInitialVehicleState 在 WS 连接建立后，将 Redis 中缓存的车辆状态推送给客户端
func pushInitialVehicleState(client *Client, vin string) {
	var data fleet.SimpleVehicleData
	if err := redispkg.GetVehicleState(vin, &data); err != nil {
		log.Printf("[WS] pushInitialVehicleState: no Redis cache for %s: %v", vin, err)
		return
	}

	if data.VIN == "" {
		data.VIN = vin
	}
	// 如果 state_output 为空，构建一个；否则同步 battery_level 防止睡眠时为0
	if data.StateOutput == nil {
		data.StateOutput = vstate.GetOutput(vin, &vstate.VehicleDataInput{
			Speed:              data.Speed,
			Gear:               data.Gear,
			ChargingState:      data.ChargingState,
			Supercharging:      data.Supercharging,
			Soc:                data.Soc,
			ChargePower:        data.ChargePower,
			MinutesToFull:      data.MinutesToFull,
			Locked:             data.Locked,
			DoorOpen:           data.DoorOpen,
			CruiseState:        data.CruiseState,
			AutosteerState:     data.AutosteerState,
			CruiseControlState: data.CruiseControlState,
		})
	} else if data.Soc > 0 {
		data.StateOutput.Charge.BatteryLevel = data.Soc
	}

	// 推送完整车辆状态
	if msg, err := json.Marshal(map[string]interface{}{
		"type": "vehicle_state",
		"data": data,
	}); err == nil {
		select {
		case client.Send <- msg:
		default:
			log.Printf("[WS] pushInitialVehicleState: send buffer full for %s", vin)
		}
	}

	// 推送在线状态
	onlineState := "offline"
	if data.Online {
		onlineState = "online"
	}
	if data.State != "" {
		onlineState = data.State
	}
	if msg, err := json.Marshal(map[string]interface{}{
		"type": "online_state",
		"data": map[string]interface{}{
			"state":  onlineState,
			"online": data.Online,
		},
	}); err == nil {
		select {
		case client.Send <- msg:
		default:
		}
	}

	// 推送轮询状态
	var pollStateData interface{}
	if err := redispkg.GetPollingState(vin, &pollStateData); err == nil {
		if msg, err := json.Marshal(map[string]interface{}{
			"type": "poll_state",
			"data": map[string]interface{}{
				"poll_state": pollStateData,
			},
		}); err == nil {
			select {
			case client.Send <- msg:
			default:
			}
		}
	}

	log.Printf("[WS] pushInitialVehicleState: pushed cached state for %s (soc=%d, state=%s)", vin, data.Soc, data.State)
}

func HandleWebSocketWithProtocol(c *gin.Context) {
	proto := c.GetHeader("Sec-WebSocket-Protocol")
	token := ""

	if proto != "" {
		parts := strings.SplitN(proto, ",", 2)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "Bearer.") {
				token = strings.TrimPrefix(p, "Bearer.")
				break
			}
		}
	}

	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token required"})
		return
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
		return
	}

	userID := claims.UserID

	var vehicles []models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND bind_status = 1", userID).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "database error"})
		return
	}

	vins := make(map[string]bool)
	for _, v := range vehicles {
		vins[v.VIN] = true
	}

	if len(vins) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "no vehicles bound"})
		return
	}

	respHeader := http.Header{}
	if proto != "" {
		respHeader.Set("Sec-WebSocket-Protocol", proto)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, respHeader)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		VINs:   vins,
		Hub:    DefaultHub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	DefaultHub.Register(client)

	go client.WritePump()
	go client.ReadPump()

	log.Printf("[WS] New connection: user_id=%d, vins=%v", userID, vins)
}
