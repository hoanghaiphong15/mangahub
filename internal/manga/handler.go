package manga

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hoanghaiphong15/mangahub/pkg/models" 
)

// Handler groups manga-related dependencies
type Handler struct {
	DB *sql.DB
}

// SearchManga searches for manga by title [cite: 814, 1293-1296]
func (h *Handler) SearchManga(c *gin.Context) {
	searchQuery := c.Query("query") // e.g., /manga?query=piece

	// Use LIKE for basic substring matching
	query := `SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE title LIKE ? LIMIT 20`
	rows, err := h.DB.Query(query, "%"+searchQuery+"%")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var results []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresStr string
		
		if err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description); err != nil {
			continue // Skip problematic rows
		}
		
		// Convert the JSON string back to a string slice
		json.Unmarshal([]byte(genresStr), &m.Genres)
		results = append(results, m)
	}

	// Return empty array instead of null if no results
	if results == nil {
		results = []models.Manga{}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// GetManga retrieves a specific manga by its ID [cite: 816, 1304-1307]
func (h *Handler) GetManga(c *gin.Context) {
	id := c.Param("id")
	var m models.Manga
	var genresStr string

	query := `SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?`
	err := h.DB.QueryRow(query, id).Scan(
		&m.ID, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	json.Unmarshal([]byte(genresStr), &m.Genres)
	c.JSON(http.StatusOK, m)
}