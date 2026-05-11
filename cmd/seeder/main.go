package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // SQLite database driver [cite: 349]
)

// Manga represents the data structure for a manga series based on project requirements [cite: 89, 91-100]
type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	Year          int      `json:"year"`
	Rating        float64  `json:"rating"`
	Popularity    int      `json:"popularity"`
}

func main() {
	// 1. Open database connection
	// Path to your SQLite database file [cite: 289]
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Read the JSON seed file
	// Ensure the file exists in the data/ directory [cite: 388]
	data, err := os.ReadFile("data/manga_seed.json")
	if err != nil {
		log.Fatal(err)
	}

	var mangas []Manga
	if err := json.Unmarshal(data, &mangas); err != nil {
		log.Fatal(err)
	}

	// 3. Insert or update records in the database
	// "INSERT OR REPLACE" ensures existing IDs are updated rather than duplicated
	query := `INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, year, rating, popularity) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, m := range mangas {
		// Convert the Genres array into a comma-separated string for SQLite storage
		genresBytes, _ := json.Marshal(m.Genres) // Chuyển slice thành chuỗi ["Action","Adventure"]
		genresStr := string(genresBytes)
		_, err := db.Exec(query, m.ID, m.Title, m.Author, genresStr, m.Status, m.TotalChapters, m.Description, m.Year, m.Rating, m.Popularity)
		if err != nil {
			fmt.Printf("Error seeding %s: %v\n", m.Title, err)
		}
	}

	fmt.Println("Database seeded successfully!")
}
