package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development. In production, check origin properly.
		return true
	},
}

// Handler handles WebSocket connections.
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeHTTP upgrades the HTTP connection to a WebSocket connection.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get user info from context (set by auth middleware)
	userID, _ := r.Context().Value("user_id").(string)
	userRole, _ := r.Context().Value("user_role").(string)

	if userID == "" {
		userID = "anonymous"
		userRole = "viewer"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := NewClient(h.hub, userID, userRole)
	h.hub.Register(client)

	// Start goroutines for reading and writing
	go h.writePump(client, conn)
	go h.readPump(client, conn)
}

// readPump pumps messages from the WebSocket connection to the hub.
func (h *Handler) readPump(client *Client, conn *websocket.Conn) {
	defer func() {
		h.hub.Unregister(client)
		_ = conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// We don't expect messages from the client in this implementation
		// If needed, handle client messages here
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (h *Handler) writePump(client *Client, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
