package domain

import (
	"testing"
	"time"
)

func TestTaskIsOverdue(t *testing.T) {
	t.Parallel()
	today := time.Date(2026, 4, 10, 15, 0, 0, 0, time.UTC)
	yesterday := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)
	tomorrow := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	t.Run("done never overdue", func(t *testing.T) {
		t.Parallel()
		task := &Task{Status: StatusDone, DueDate: &yesterday}
		if task.IsOverdue(today) {
			t.Fatal("done task should not be overdue")
		}
	})

	t.Run("no due date", func(t *testing.T) {
		t.Parallel()
		task := &Task{Status: StatusTodo, DueDate: nil}
		if task.IsOverdue(today) {
			t.Fatal("nil due date")
		}
	})

	t.Run("due yesterday open", func(t *testing.T) {
		t.Parallel()
		task := &Task{Status: StatusTodo, DueDate: &yesterday}
		if !task.IsOverdue(today) {
			t.Fatal("expected overdue")
		}
	})

	t.Run("due today not overdue", func(t *testing.T) {
		t.Parallel()
		due := time.Date(2026, 4, 10, 23, 59, 0, 0, time.UTC)
		task := &Task{Status: StatusInProgress, DueDate: &due}
		if task.IsOverdue(today) {
			t.Fatal("due today should not be overdue")
		}
	})

	t.Run("due tomorrow", func(t *testing.T) {
		t.Parallel()
		task := &Task{Status: StatusTodo, DueDate: &tomorrow}
		if task.IsOverdue(today) {
			t.Fatal("future due")
		}
	})
}
