package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of WebSocket message.
type MessageType string

const (
	// MessageTypeProxyStatus indicates proxy status update.
	MessageTypeProxyStatus MessageType = "proxy_status"
	// MessageTypeDeviceUpdate indicates device list update.
	MessageTypeDeviceUpdate MessageType = "device_update"
	// MessageTypeMetrics indicates metrics update.
	MessageTypeMetrics MessageType = "metrics"
	// MessageTypeLog indicates new log entry.
	MessageTypeLog MessageType = "log"
	// MessageTypeAudit indicates audit log entry.
	MessageTypeAudit MessageType = "audit"
)

// Message represents a WebSocket message.
type Message struct {
	Type      MessageType     `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// Client represents a WebSocket client connection.
type Client struct {
	ID       string
	Hub      *Hub
	Send     chan Message
	UserID   string // For authentication
	UserRole string // For RBAC
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client's send channel is full, close it
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msgType MessageType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	message := Message{
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      jsonData,
	}

	h.broadcast <- message
	return nil
}

// Register registers a new client.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, userID, userRole string) *Client {
	return &Client{
		ID:       uuid.New().String(),
		Hub:      hub,
		Send:     make(chan Message, 256),
		UserID:   userID,
		UserRole: userRole,
	}
}
