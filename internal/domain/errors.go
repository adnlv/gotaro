package domain

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrDuplicateEmail = errors.New("email already registered")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrNotFound       = errors.New("not found")
)
