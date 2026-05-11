package user

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hoanghaiphong15/mangahub/internal/tcp"
)

// Handler groups user library dependencies
type Handler struct {
	DB           *sql.DB
	TCPBroadcast chan tcp.ProgressUpdate // Channel to send progress updates to the TCP server
}

// DTOs for incoming requests
type AddLibraryRequest struct {
	MangaID string `json:"manga_id" binding:"required"`
	Status  string `json:"status" binding:"required"` // e.g., "reading", "completed", "plan_to_read"
}

type UpdateProgressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"required"`
}

// AddLibrary adds a manga to the user's personal tracking list [cite: 817, 1312-1316]
func (h *Handler) AddLibrary(c *gin.Context) {
	// Extract the user ID set by the JWTMiddleware
	userID := c.GetString("user_id")

	var req AddLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Insert into user_progress. If it already exists, we could return a conflict or update it.
	// For simplicity, we'll assume it's a fresh add.
	query := `INSERT INTO user_progress (user_id, manga_id, current_chapter, status, updated_at) 
			  VALUES (?, ?, ?, ?, ?)`

	_, err := h.DB.Exec(query, userID, req.MangaID, 0, req.Status, time.Now())
	if err != nil {
		// If UNIQUE constraint fails, it means it's already in the library
		c.JSON(http.StatusConflict, gin.H{"error": "Manga is already in your library"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Added to library successfully",
		"manga_id": req.MangaID,
		"status":   req.Status,
	})
}

// UpdateProgress updates the current chapter the user is reading [cite: 819, 1322-1326]
func (h *Handler) UpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")

	var req UpdateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	query := `UPDATE user_progress SET current_chapter = ?, updated_at = ? 
			  WHERE user_id = ? AND manga_id = ?`

	result, err := h.DB.Exec(query, req.CurrentChapter, time.Now(), userID, req.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found in your library"})
		return
	}

	// 3. Trigger the TCP Broadcast! [cite: 1431]
	// Create the update message
	updateMsg := tcp.ProgressUpdate{
		UserID:    userID,
		MangaID:   req.MangaID,
		Chapter:   req.CurrentChapter,
		Timestamp: time.Now().Unix(),
	}

	// Use a non-blocking select so the HTTP request doesn't hang
	// if the TCP server is overwhelmed or turned off.
	select {
	case h.TCPBroadcast <- updateMsg:
		// Message successfully sent to the TCP server's channel!
	default:
		// Channel is full or unavailable; silently continue
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Progress updated successfully",
		"current_chapter": req.CurrentChapter,
	})
}

func (h *Handler) GetLibrary(c *gin.Context) {

	userID := c.GetString("user_id")

	query := `
		SELECT 
			m.id,
			m.title,
			m.author,
			up.current_chapter,
			up.status,
			up.updated_at
		FROM user_progress up
		JOIN manga m ON up.manga_id = m.id
		WHERE up.user_id = ?
	`

	rows, err := h.DB.Query(query, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database error",
		})
		return
	}

	defer rows.Close()

	type LibraryItem struct {
		MangaID        string    `json:"manga_id"`
		Title          string    `json:"title"`
		Author         string    `json:"author"`
		CurrentChapter int       `json:"current_chapter"`
		Status         string    `json:"status"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	var library []LibraryItem

	for rows.Next() {

		var item LibraryItem

		err := rows.Scan(
			&item.MangaID,
			&item.Title,
			&item.Author,
			&item.CurrentChapter,
			&item.Status,
			&item.UpdatedAt,
		)

		if err != nil {
			continue
		}

		library = append(library, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"library": library,
	})
}
