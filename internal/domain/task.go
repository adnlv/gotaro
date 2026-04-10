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

// IsOverdue reports whether the task is still open and its due date is strictly before
// the calendar day of asOf (compared in UTC). Done and archived tasks are never overdue.
func (t *Task) IsOverdue(asOf time.Time) bool {
	if t.Status == StatusDone || t.IsArchived() {
		return false
	}
	if t.DueDate == nil {
		return false
	}
	d := t.DueDate.UTC()
	n := asOf.UTC()
	dueDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	return dueDay.Before(today)
}

// IsArchived is true when the task has been archived (soft-hidden from normal lists).
func (t *Task) IsArchived() bool {
	return t.ArchivedAt != nil && !t.ArchivedAt.IsZero()
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
