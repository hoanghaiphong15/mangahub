package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"

	grpc_internal "github.com/hoanghaiphong15/mangahub/internal/grpc"
	pb "github.com/hoanghaiphong15/mangahub/pkg/proto"

	"github.com/gin-gonic/gin"
	"github.com/hoanghaiphong15/mangahub/internal/auth"
	"github.com/hoanghaiphong15/mangahub/internal/manga"
	"github.com/hoanghaiphong15/mangahub/internal/tcp"
	"github.com/hoanghaiphong15/mangahub/internal/udp"
	"github.com/hoanghaiphong15/mangahub/internal/user"
	"github.com/hoanghaiphong15/mangahub/internal/websocket"
	"github.com/hoanghaiphong15/mangahub/pkg/database"
	"github.com/gin-contrib/cors"
)

// APIServer holds the core dependencies for the HTTP server [cite: 806-811]
type APIServer struct {
	Router    *gin.Engine
	Database  *sql.DB
	JWTSecret string
	TCPServer *tcp.ProgressSyncServer
	UDPServer *udp.NotificationServer
	WSHub     *websocket.ChatHub
}

func main() {
	// 1. Initialize the SQLite Database
	// Use environment variable for DB path, default to local file for standard development
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data.db"
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 2. Start TCP Server
	tcpServer := tcp.NewServer(":9090")

	go func() {
		if err := tcpServer.Start(); err != nil {
			log.Fatalf("Failed to start TCP server: %v", err)
		}
	}()

	// 3. Start UDP Server
	udpServer := udp.NewServer(":9091")

	go func() {
		if err := udpServer.Start(); err != nil {
			log.Fatalf("Failed to start UDP server: %v", err)
		}
	}()

	// 4. Start WebSocket Chat Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	// 4. Start gRPC Internal Service Server on port 9092
	grpcListener, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Create our implementation and register it
	mangaGrpcService := &grpc_internal.Server{
		DB:           db,
		TCPBroadcast: tcpServer.Broadcast,
	}
	pb.RegisterMangaServiceServer(grpcServer, mangaGrpcService)

	go func() {
		log.Printf("Starting gRPC Internal Service on grpc://localhost:9092")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()
	// 5. Setup API Server
	server := &APIServer{
		Router:    gin.Default(),
		Database:  db,
		JWTSecret: getJWTSecret(),
		TCPServer: tcpServer,
		UDPServer: udpServer,
		WSHub:     wsHub,
	}

	server.Router.Use(cors.Default())

	// 6. Register Routes
	server.setupRoutes()

	// 7. Start HTTP Server
	port := ":8080"

	log.Printf("Starting HTTP API Server on http://localhost%s", port)

	if err := server.Router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupRoutes configures all the HTTP endpoints for the application
func (s *APIServer) setupRoutes() {
	// Initialize the Auth Handler
	authHandler := &auth.Handler{
		DB:        s.Database,
		JWTSecret: []byte(s.JWTSecret),
	}

	// Simple health check endpoint
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "Online", "service": "HTTP API"})
	})

	// Mount the Auth routes [cite: 812-813]
	authGroup := s.Router.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	// Placeholder groups for our future endpoint
	// Initialize the Manga Handler
	mangaHandler := &manga.Handler{
		DB: s.Database,
	}

	// Start background tasks to fetch data from MangaDex and practice scraping
	log.Println(">>> Initializing Data Collection Tasks...")
	go mangaHandler.FetchFromMangaDex() // run in background to fetch manga data from MangaDex API
	go mangaHandler.PracticeScraping()  // run in background to perform educational web scraping practice

	// Mount the Manga routes [cite: 814-816]
	mangaGroup := s.Router.Group("/manga")
	{
		mangaGroup.GET("", mangaHandler.SearchManga)
		mangaGroup.GET("/:id", mangaHandler.GetManga)
		mangaGroup.POST("/search", mangaHandler.AdvancedSearch)
	}

	// Initialize the User Handler
	userHandler := &user.Handler{
		DB:           s.Database,
		TCPBroadcast: s.TCPServer.Broadcast, // Pass the TCP server's broadcast channel to the user handler
	}

	// User Library & Progress routes (Protected by JWT)
	users := s.Router.Group("/users")
	users.Use(auth.JWTMiddleware([]byte(s.JWTSecret)))
	{
		users.POST("/library", userHandler.AddLibrary)
		users.GET("/library", userHandler.GetLibrary)
		users.PUT("/progress", userHandler.UpdateProgress)
	}

	// Temporary Admin route to test UDP Broadcasts
	s.Router.POST("/admin/notify", func(c *gin.Context) {
		var notif udp.Notification
		if err := c.ShouldBindJSON(&notif); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
			return
		}

		// Blast the notification via UDP!
		s.UDPServer.Broadcast(notif)
		c.JSON(http.StatusOK, gin.H{"message": "Notification broadcasted!"})
	})

	// WebSocket Chat Route [cite: 872-873, 1372-1382]
	s.Router.GET("/ws", func(c *gin.Context) {
		websocket.ServeWs(s.WSHub, c)
	})
}

// getJWTSecret retrieves the secret key from environment variables or uses a default
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("Warning: JWT_SECRET environment variable not set. Using default secret.")
		return "mangahub_super_secret_key_123"
	}
	return secret
}
