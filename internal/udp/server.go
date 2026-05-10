package udp

import (
	"encoding/json"
	"log"
	"net"
	"sync"
)

// Notification matches the project spec [cite: 858-867]
type Notification struct {
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// NotificationServer handles UDP subscriptions and broadcasts [cite: 853-857]
type NotificationServer struct {
	Port    string
	Clients map[string]*net.UDPAddr // Map to store unique client addresses
	mu      sync.RWMutex
}

// NewServer initializes the UDP server
func NewServer(port string) *NotificationServer {
	return &NotificationServer{
		Port:    port,
		Clients: make(map[string]*net.UDPAddr),
	}
}

// Start opens the UDP listener and waits for subscriptions
func (s *NotificationServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.Port)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("Starting UDP Notification Server on udp://localhost%s\n", s.Port)

	buf := make([]byte, 1024)
	for {
		// Read incoming UDP packets
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP Read error: %v", err)
			continue
		}

		msg := string(buf[:n])
		
		// If a client sends "subscribe", add them to our list [cite: 1358-1361]
		if msg == "subscribe" {
			s.mu.Lock()
			s.Clients[clientAddr.String()] = clientAddr
			s.mu.Unlock()
			log.Printf("UDP Client subscribed: %s", clientAddr.String())
			
			// Send a quick confirmation back
			conn.WriteToUDP([]byte(`{"message": "Successfully subscribed to notifications!"}`+"\n"), clientAddr)
		}
	}
}

// Broadcast sends a notification to all subscribed clients [cite: 1363-1368]
func (s *NotificationServer) Broadcast(notification Notification) {
	addr, _ := net.ResolveUDPAddr("udp", s.Port)
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Failed to create UDP broadcast connection: %v", err)
		return
	}
	defer conn.Close()

	data, err := json.Marshal(notification)
	if err != nil {
		return
	}
	data = append(data, '\n')

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Blast the message to every registered client
	for _, clientAddr := range s.Clients {
		_, err := conn.WriteToUDP(data, clientAddr)
		if err != nil {
			log.Printf("Failed to send UDP to %s: %v", clientAddr.String(), err)
		}
	}
}