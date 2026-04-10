package web

import (
	"strings"
	"testing"
	"time"
)

func TestFormatTaskCreated(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 4, 10, 14, 7, 0, 0, time.UTC)
	got := formatTaskCreated(ts)
	if !strings.Contains(got, "Apr") || !strings.Contains(got, "2026") || !strings.Contains(got, "2:07 PM") {
		t.Fatalf("unexpected: %q", got)
	}
	if !strings.HasSuffix(got, " UTC") {
		t.Fatalf("expected UTC suffix: %q", got)
	}
}

func TestFormatTaskDue(t *testing.T) {
	t.Parallel()
	ref := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	today := time.Date(2026, 4, 10, 8, 0, 0, 0, time.FixedZone("X", 5*3600)) // same UTC day
	if got := formatTaskDue(today, ref); !strings.Contains(got, "Today") {
		t.Fatalf("expected Today: %q", got)
	}
	tomorrow := ref.AddDate(0, 0, 1)
	if got := formatTaskDue(tomorrow, ref); !strings.Contains(got, "Tomorrow") {
		t.Fatalf("expected Tomorrow: %q", got)
	}
	past := ref.AddDate(0, 0, -5)
	if got := formatTaskDue(past, ref); strings.Contains(got, "Today") {
		t.Fatalf("unexpected Today: %q", got)
	}
}
