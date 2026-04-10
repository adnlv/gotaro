package domain

import "time"

type User struct {
	ID           uint64
	Email        string
	PasswordHash string
	RegisteredAt time.Time
	UpdatedAt    time.Time
}
