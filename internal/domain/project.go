package domain

import "strings"

// Project is a user-defined bucket for tasks (category).
type Project struct {
	ID     uint64
	UserID uint64
	Name   string
}

// NormalizeProjectName trims and collapses internal space for comparison and storage keys.
func NormalizeProjectName(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}
