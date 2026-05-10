package models

// Manga represents a comic series in the database
type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"` // Stored as JSON string in SQLite
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url,omitempty"`
}