package domain

import (
	"testing"
	"time"
)

func TestTaskIsArchived(t *testing.T) {
	t.Parallel()
	var zero time.Time
	if (&Task{ArchivedAt: nil}).IsArchived() {
		t.Fatal("nil")
	}
	if (&Task{ArchivedAt: &zero}).IsArchived() {
		t.Fatal("zero time")
	}
	now := time.Now().UTC()
	if !(&Task{ArchivedAt: &now}).IsArchived() {
		t.Fatal("expected archived")
	}
}
