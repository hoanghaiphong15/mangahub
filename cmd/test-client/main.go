package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	// Connect to your TCP server
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		log.Fatalf("Failed to connect to TCP server: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Successfully connected to MangaHub TCP Sync Server!")
	fmt.Println("Waiting for progress updates...")

	// Listen for incoming messages in an infinite loop
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		fmt.Println("📩 NEW UPDATE RECEIVED:", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Connection lost: %v", err)
	}
}