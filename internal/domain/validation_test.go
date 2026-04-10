package domain

import (
	"strings"
	"testing"
	"time"
)

func TestValidateEmail(t *testing.T) {
	t.Parallel()
	if err := ValidateEmail(""); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateEmail("not-an-email"); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateEmail("a@b.co"); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidatePassword("longenough"); err != nil {
		t.Fatal(err)
	}
}

func TestParseTagList(t *testing.T) {
	t.Parallel()
	got, err := ParseTagList("  a , B , a  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %#v", got)
	}
	if _, err := ParseTagList(strings.Repeat("x", maxTagNameLen+1)); err == nil {
		t.Fatal("expected error for long tag")
	}
}

func TestValidateTaskInput(t *testing.T) {
	t.Parallel()
	if err := ValidateTaskInput("", nil, nil); err == nil {
		t.Fatal("expected error")
	}
	d := strings.Repeat("x", maxDescLen+1)
	if err := ValidateTaskInput("ok", &d, nil); err == nil {
		t.Fatal("expected error")
	}
	due := time.Time{}
	if err := ValidateTaskInput("ok", nil, &due); err == nil {
		t.Fatal("expected error for zero due")
	}
}

func TestValidateSortField(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"created_at", "due_date", "priority", "status", ""} {
		if _, err := ValidateSortField(f); err != nil {
			t.Fatalf("%q: %v", f, err)
		}
	}
	if _, err := ValidateSortField("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestStatusPriorityFromString(t *testing.T) {
	t.Parallel()
	if s, ok := StatusFromString("todo"); !ok || s != StatusTodo {
		t.Fatal(s, ok)
	}
	if p, ok := PriorityFromString("high"); !ok || p != PriorityHigh {
		t.Fatal(p, ok)
	}
}
