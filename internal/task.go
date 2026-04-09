package internal

import "time"

type Status int

const (
	StatusTodo Status = iota
	StatusInProgress
	StatusDone
)

type Priority int

const (
	PriorityLow Priority = iota - 1
	PriorityMedium
	PriorityHigh
)

type Task struct {
	ID          uint64     `json:"id"`
	Author      *User      `json:"author"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Tags        []*Tag     `json:"tags,omitempty"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}
