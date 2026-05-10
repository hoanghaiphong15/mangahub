package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Handler groups our auth-related dependencies
type Handler struct {
	DB        *sql.DB
	JWTSecret []byte
}

// Data Transfer Objects (DTOs) for incoming requests
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register creates a new user account [cite: 812, 1269-1272]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 1. Hash the password 
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// 2. Generate a unique ID and insert into the database [cite: 1271]
	userID := "usr_" + uuid.New().String()[:8] // Shortened for cleaner IDs
	
	query := `INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`
	_, err = h.DB.Exec(query, userID, req.Username, string(hashedPassword))
	if err != nil {
		// In SQLite, an error here usually means the UNIQUE constraint on username failed
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Account created successfully",
		"user_id":  userID,
		"username": req.Username,
	})
}

// Login authenticates a user and returns a JWT token [cite: 813, 1282-1286]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 1. Fetch user from database
	var userID, storedHash string
	query := `SELECT id, password_hash FROM users WHERE username = ?`
	err := h.DB.QueryRow(query, req.Username).Scan(&userID, &storedHash)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 2. Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 3. Generate JWT Token [cite: 1284]
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	tokenString, err := token.SignedString(h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
	})
}