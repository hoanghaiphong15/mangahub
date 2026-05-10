package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In production, check the origin to prevent CSRF. For this project, allow all.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWs handles the HTTP request and upgrades it to a WebSocket connection
func ServeWs(hub *ChatHub, c *gin.Context) {
	// Upgrade initial GET request to a websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade websocket:", err)
		return
	}

	// Register the new connection with the hub
	hub.Register <- conn

	// Make sure we unregister when this function exits
	defer func() {
		hub.Unregister <- conn
	}()

	// Infinite loop to read incoming messages from this specific client [cite: 1388-1392]
	for {
		var msg ChatMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break // Break the loop and disconnect if there's an error
		}

		// Ensure the timestamp is set securely on the server side
		msg.Timestamp = time.Now().Unix()

		// If this is the user's first message, we can associate their UserID with the connection
		hub.mu.Lock()
		if hub.Clients[conn] == "" {
			hub.Clients[conn] = msg.UserID
		}
		hub.mu.Unlock()

		// Send the message to the Hub to be broadcasted to everyone
		hub.Broadcast <- msg
	}
}