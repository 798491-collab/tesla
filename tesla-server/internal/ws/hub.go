package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID uint64
	VINs   map[string]bool
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	vinClients map[string]map[*Client]bool
	mu         sync.RWMutex
	register   chan *Client
	unregister chan *Client
}

var DefaultHub *Hub

func InitHub() {
	DefaultHub = &Hub{
		clients:    make(map[*Client]bool),
		vinClients: make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go DefaultHub.Run()
	log.Printf("[WS] Hub started")
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			for vin := range client.VINs {
				if h.vinClients[vin] == nil {
					h.vinClients[vin] = make(map[*Client]bool)
				}
				h.vinClients[vin][client] = true
			}
			h.mu.Unlock()
			log.Printf("[WS] Client registered (user_id=%d, vins=%d)", client.UserID, len(client.VINs))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				for vin := range client.VINs {
					if subs, ok := h.vinClients[vin]; ok {
						delete(subs, client)
						if len(subs) == 0 {
							delete(h.vinClients, vin)
						}
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[WS] Client unregistered (user_id=%d)", client.UserID)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

type WSMessage struct {
	Type string      `json:"type"`
	VIN  string      `json:"vin"`
	Data interface{} `json:"data"`
	Time int64       `json:"time"`
}

func (h *Hub) BroadcastToVIN(vin string, msgType string, data interface{}) {
	msg := WSMessage{
		Type: msgType,
		VIN:  vin,
		Data: data,
		Time: time.Now().UnixMilli(),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.vinClients[vin]
	if !ok {
		h.mu.RUnlock()
		return
	}

	var targets []*Client
	for client := range clients {
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	sent := 0
	dropped := 0
	for _, client := range targets {
		select {
		case client.Send <- payload:
			sent++
		default:
			dropped++
			go h.Unregister(client)
		}
	}

	if dropped > 0 {
		log.Printf("[WS] Broadcast to VIN %s: type=%s, sent=%d, dropped=%d", vin, msgType, sent, dropped)
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for msg := range c.Send {
		c.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var pingMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg, &pingMsg); err == nil && pingMsg.Type == "ping" {
			c.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			c.Conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		}
	}
}

func BroadcastVehicleState(vin string, data interface{}) {
	if DefaultHub != nil {
		DefaultHub.BroadcastToVIN(vin, "vehicle_state", data)
	}
}

func BroadcastRealtimeUpdate(vin string, data interface{}) {
	if DefaultHub != nil {
		DefaultHub.BroadcastToVIN(vin, "realtime_update", data)
	}
}

func BroadcastStateUpdate(vin string, data interface{}) {
	if DefaultHub != nil {
		DefaultHub.BroadcastToVIN(vin, "state_update", data)
	}
}

func BroadcastOnlineState(vin string, onlineState string, online bool) {
	if DefaultHub != nil {
		msgData := map[string]interface{}{
			"state":  onlineState,
			"online": online,
		}
		DefaultHub.BroadcastToVIN(vin, "online_state", msgData)
	}
}

func BroadcastPollState(vin string, pollState string) {
	if DefaultHub != nil {
		DefaultHub.BroadcastToVIN(vin, "poll_state", map[string]interface{}{
			"poll_state": pollState,
		})
	}
}

func BroadcastCommandState(vin string, cmdState string, command string, latencyMs int64) {
	if DefaultHub != nil {
		DefaultHub.BroadcastToVIN(vin, "command_state", map[string]interface{}{
			"command_state": cmdState,
			"last_command":  command,
			"latency_ms":    latencyMs,
		})
	}
}
