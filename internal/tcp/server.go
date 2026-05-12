package tcp

import (
	"encoding/json"
	"log"
	"net"
	"sync"
)

// ProgressUpdate matches the exact JSON structure from your project spec [cite: 834-844]
type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

// ProgressSyncServer handles concurrent TCP connections and broadcasting [cite: 828-833]
type ProgressSyncServer struct {
	Port        string
	Connections map[string]net.Conn
	Broadcast   chan ProgressUpdate
	mu          sync.RWMutex // Mutex ensures safe access to the Connections map across goroutines
}

// NewServer initializes the TCP server
func NewServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer{
		Port:        port,
		Connections: make(map[string]net.Conn),
		Broadcast:   make(chan ProgressUpdate, 100), // Buffered channel to prevent deadlock
	}
}

// ConnectionCount returns the number of active TCP connections (thread-safe)
func (s *ProgressSyncServer) ConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Connections)
}

// Start opens the TCP listener and begins accepting clients [cite: 932]
func (s *ProgressSyncServer) Start() error {
	listener, err := net.Listen("tcp", s.Port)
	if err != nil {
		return err
	}
	log.Printf("Starting TCP Sync Server on tcp://localhost%s\n", s.Port)

	// Start the broadcaster worker in the background
	go s.handleBroadcasts()

	// Infinite loop to accept incoming connections [cite: 845]
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Spin up a new goroutine for every single client
		go s.handleConnection(conn)
	}
}

// handleConnection manages the lifecycle of a single client connection
func (s *ProgressSyncServer) handleConnection(conn net.Conn) {
	connID := conn.RemoteAddr().String() // Use IP:Port as a temporary ID

	// Safely add the connection to our map
	s.mu.Lock()
	s.Connections[connID] = conn
	s.mu.Unlock()

	log.Printf("TCP Client connected: %s", connID)

	// Defer cleanup for when the client disconnects [cite: 935, 1350]
	defer func() {
		s.mu.Lock()
		delete(s.Connections, connID)
		s.mu.Unlock()
		conn.Close()
		log.Printf("TCP Client disconnected: %s", connID)
	}()

	// Keep the connection alive by reading from it until it closes
	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			break // Break the loop if the client disconnects or an error occurs
		}
	}
}

// handleBroadcasts listens for updates on the channel and sends them to all clients [cite: 846, 1346-1349]
func (s *ProgressSyncServer) handleBroadcasts() {
	for update := range s.Broadcast {
		// Convert the struct to JSON bytes [cite: 848, 933]
		data, err := json.Marshal(update)
		if err != nil {
			log.Printf("TCP Sync: Failed to marshal update: %v", err)
			continue
		}
		data = append(data, '\n') // Add a newline to separate messages for the client

		// Safely loop through all connected clients and send the message
		s.mu.RLock()
		for id, conn := range s.Connections {
			_, err := conn.Write(data)
			if err != nil {
				log.Printf("TCP Sync: Failed to write to client %s: %v", id, err)
			}
		}
		s.mu.RUnlock()
	}
}
