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

// SearchFilters matches the exact struct from your project spec [cite: 1107-1118]
type SearchFilters struct {
	Keyword   string   `json:"keyword"`
	Genres    []string `json:"genres"`
	Status    string   `json:"status"`
	YearRange [2]int   `json:"year_range"`
	Rating    float64  `json:"rating"`
	SortBy    string   `json:"sort_by"` // "popularity", "rating", "recent"
}

// SearchManga searches for manga by title [cite: 814, 1293-1296]
func (h *Handler) SearchManga(c *gin.Context) {
	searchQuery := c.Query("query") // e.g., /manga?query=piece

	// Use LIKE for basic substring matching
	query := `SELECT id, title, author, genres, status, total_chapters, description, year, rating, popularity FROM manga WHERE title LIKE ? LIMIT 300`
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

		if err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description, &m.Year, &m.Rating, &m.Popularity); err != nil {
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

	query := `SELECT id, title, author, genres, status, total_chapters, description, year, rating, popularity FROM manga WHERE id = ?`
	err := h.DB.QueryRow(query, id).Scan(
		&m.ID, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description, &m.Year, &m.Rating, &m.Popularity,
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

// AdvancedSearch handles complex filtering and full-text search [cite: 1435-1443]
func (h *Handler) AdvancedSearch(c *gin.Context) {
	var filters SearchFilters
	if err := c.ShouldBindJSON(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// 1. Start the base query
	query := `SELECT id, title, author, genres, status, total_chapters, description, year, rating, popularity FROM manga WHERE 1=1`
	var args []interface{}

	// 2. Dynamically build the WHERE clause based on provided filters
	if filters.Keyword != "" {
		// Full-text search on title and description [cite: 1442]
		query += ` AND (title LIKE ? OR description LIKE ?)`
		args = append(args, "%"+filters.Keyword+"%", "%"+filters.Keyword+"%")
	}

	if filters.Status != "" {
		query += ` AND status = ?`
		args = append(args, filters.Status)
	}

	if filters.Rating > 0 {
		query += ` AND rating >= ?`
		args = append(args, filters.Rating)
	}

	if filters.YearRange[0] > 0 && filters.YearRange[1] > 0 {
		query += ` AND year BETWEEN ? AND ?`
		args = append(args, filters.YearRange[0], filters.YearRange[1])
	}

	// Filter by Genres (SQLite JSON search workaround using LIKE)
	for _, genre := range filters.Genres {
		query += ` AND genres LIKE ?`
		args = append(args, "%"+genre+"%")
	}

	// 3. Add Sorting logic [cite: 1116-1117]
	switch filters.SortBy {
	case "popularity":
		query += ` ORDER BY popularity DESC`
	case "rating":
		query += ` ORDER BY rating DESC`
	case "recent":
		query += ` ORDER BY year DESC`
	default:
		query += ` ORDER BY title ASC`
	}

	// Limit results
	query += ` LIMIT 50`

	// 4. Execute the complex query
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresStr string

		if err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description, &m.Year, &m.Rating, &m.Popularity); err != nil {
			continue
		}

		json.Unmarshal([]byte(genresStr), &m.Genres)
		results = append(results, m)
	}

	if results == nil {
		results = []models.Manga{}
	}

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}
