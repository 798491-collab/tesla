package ws

import (
	"log"
	"net/http"
	"strings"
	"tesla-server/internal/auth"
	"tesla-server/internal/database"
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

	log.Printf("[WS] New VIN connection: user_id=%d, vin=%s", claims.UserID, vin)
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
