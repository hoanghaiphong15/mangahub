package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:9091")
	if err != nil {
		log.Fatal(err)
	}

	// Connect to the server
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// 1. Send the "subscribe" message
	_, err = conn.Write([]byte("subscribe"))
	if err != nil {
		log.Fatal("Failed to send subscribe message:", err)
	}
	fmt.Println("✅ Sent subscription request to UDP Server...")

	// 2. Listen for notifications forever
	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal("Error reading from UDP:", err)
		}
		fmt.Println("🔔 NEW NOTIFICATION:", string(buf[:n]))
	}
}