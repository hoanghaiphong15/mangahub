package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite3 driver
)

// InitDB initializes the SQLite database connection and creates tables if they don't exist
func InitDB(dataSourceName string) (*sql.DB, error) {
	// Open connection to SQLite database
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create the necessary tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("Database connection established and tables verified.")
	return db, nil
}

func createTables(db *sql.DB) error {
	// Schema for users [cite: 902-907]
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE,
		password_hash TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Schema for manga [cite: 908-916]
	createMangaTable := `
	CREATE TABLE IF NOT EXISTS manga (
		id TEXT PRIMARY KEY,
		title TEXT,
		author TEXT,
		genres TEXT,
		status TEXT,
		total_chapters INTEGER,
		description TEXT,
		year INTEGER DEFAULT 2000,
		rating REAL DEFAULT 0.0,
		popularity INTEGER DEFAULT 0
	);`

	// Schema for user progress [cite: 917-923]
	createUserProgressTable := `
	CREATE TABLE IF NOT EXISTS user_progress (
		user_id TEXT,
		manga_id TEXT,
		current_chapter INTEGER,
		status TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, manga_id)
	);`

	// Execute the statements
	statements := []string{createUsersTable, createMangaTable, createUserProgressTable}
	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}