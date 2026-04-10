package domain

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
	ID          uint64
	UserID      uint64
	Title       string
	Description *string
	Tags        []Tag
	Status      Status
	Priority    Priority
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
	DueDate     *time.Time
}

func (s Status) String() string {
	switch s {
	case StatusTodo:
		return "todo"
	case StatusInProgress:
		return "in_progress"
	case StatusDone:
		return "done"
	default:
		return "unknown"
	}
}

func StatusFromString(v string) (Status, bool) {
	switch v {
	case "todo", "":
		return StatusTodo, true
	case "in_progress":
		return StatusInProgress, true
	case "done":
		return StatusDone, true
	default:
		return 0, false
	}
}

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	default:
		return "medium"
	}
}

func PriorityFromString(v string) (Priority, bool) {
	switch v {
	case "low":
		return PriorityLow, true
	case "medium", "":
		return PriorityMedium, true
	case "high":
		return PriorityHigh, true
	default:
		return 0, false
	}
}
