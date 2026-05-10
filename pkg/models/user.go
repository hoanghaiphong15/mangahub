package models

import "time"

// User represents a registered reader in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never send the hash to the client
	CreatedAt    time.Time `json:"created_at"`
}