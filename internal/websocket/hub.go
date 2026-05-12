package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// ChatMessage represents a single message in the chat room [cite: 883-887]
type ChatMessage struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// ChatHub manages active WebSocket connections and broadcasts messages [cite: 874-882]
type ChatHub struct {
	Clients    map[*websocket.Conn]string // Maps connection to UserID
	Broadcast  chan ChatMessage
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	mu         sync.RWMutex
}

// NewHub initializes the chat room
func NewHub() *ChatHub {
	return &ChatHub{
		Clients:    make(map[*websocket.Conn]string),
		Broadcast:  make(chan ChatMessage),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
	}
}

// ClientCount returns the number of connected WebSocket clients (thread-safe)
func (h *ChatHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}

// Run starts the infinite loop to handle registrations and broadcasts [cite: 888-891]
func (h *ChatHub) Run() {
	log.Println("Starting WebSocket Chat Hub...")
	for {
		select {
		case conn := <-h.Register:
			// A new user joined
			h.mu.Lock()
			h.Clients[conn] = "" // UserID will be set during the first message/auth
			h.mu.Unlock()
			log.Println("WebSocket Client connected")

		case conn := <-h.Unregister:
			// A user left
			h.mu.Lock()
			if _, ok := h.Clients[conn]; ok {
				delete(h.Clients, conn)
				conn.Close()
				log.Println("WebSocket Client disconnected")
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			log.Printf(" Broadcasting message from %s: %s", message.Username, message.Message)
			// A new message arrived, send it to everyone
			var toRemove []*websocket.Conn
			h.mu.RLock()
			for conn := range h.Clients {
				if err := conn.WriteJSON(message); err != nil {
					log.Printf("WebSocket Write Error: %v", err)
					toRemove = append(toRemove, conn)
				}
			}
			h.mu.RUnlock()
			// Remove failed connections under write lock (safe)
			if len(toRemove) > 0 {
				h.mu.Lock()
				for _, conn := range toRemove {
					conn.Close()
					delete(h.Clients, conn)
				}
				h.mu.Unlock()
			}
		}
	}
}
