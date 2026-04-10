package domain

import "time"

type Session struct {
	Token      string
	CSRFToken  string
	User       *User
	ClientIP   string
	UserAgent  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ExpiresAt  time.Time
}
