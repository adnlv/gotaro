package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	minPasswordLen = 8
	maxTitleLen    = 500
	maxDescLen     = 10000
	maxTagNameLen  = 64
)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	return nil
}

func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLen {
		return fmt.Errorf("%w: password must be at least %d characters", ErrInvalidInput, minPasswordLen)
	}
	return nil
}

func ValidateCredentials(email, password string) error {
	if err := ValidateEmail(email); err != nil {
		return err
	}
	return ValidatePassword(password)
}

func NormalizeTagName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func ParseTagList(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		n := NormalizeTagName(p)
		if n == "" {
			continue
		}
		if utf8.RuneCountInString(n) > maxTagNameLen {
			return nil, fmt.Errorf("%w: tag name too long", ErrInvalidInput)
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out, nil
}

func ValidateTaskInput(title string, description *string, due *time.Time) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return fmt.Errorf("%w: title too long", ErrInvalidInput)
	}
	if description != nil && *description != "" {
		if utf8.RuneCountInString(*description) > maxDescLen {
			return fmt.Errorf("%w: description too long", ErrInvalidInput)
		}
	}
	if due != nil && due.IsZero() {
		return fmt.Errorf("%w: invalid due date", ErrInvalidInput)
	}
	return nil
}

func ValidateSortField(field string) (string, error) {
	switch strings.TrimSpace(field) {
	case "", "created_at":
		return "created_at", nil
	case "due_date":
		return "due_date", nil
	case "priority":
		return "priority", nil
	case "status":
		return "status", nil
	default:
		return "", fmt.Errorf("%w: invalid sort field", ErrInvalidInput)
	}
}

func ValidateSortDir(dir string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(dir)) {
	case "", "desc":
		return "DESC", nil
	case "asc":
		return "ASC", nil
	default:
		return "", fmt.Errorf("%w: invalid sort direction", ErrInvalidInput)
	}
}
