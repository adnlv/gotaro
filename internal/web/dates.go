package web

import (
	"time"
)

// formatTaskCreated renders a full timestamp in UTC for list rows.
func formatTaskCreated(t time.Time) string {
	return t.UTC().Format("Mon, Jan 2, 2006 · 3:04 PM") + " UTC"
}

// formatTaskDue renders the calendar due date with a short relative hint when it is today, tomorrow, or yesterday (UTC).
func formatTaskDue(d, reference time.Time) string {
	du := d.UTC()
	ref := reference.UTC()
	dueDay := time.Date(du.Year(), du.Month(), du.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)

	s := du.Format("Mon, Jan 2, 2006")
	switch {
	case dueDay.Equal(today):
		return s + " · Today"
	case dueDay.Equal(today.AddDate(0, 0, 1)):
		return s + " · Tomorrow"
	case dueDay.Equal(today.AddDate(0, 0, -1)):
		return s + " · Yesterday"
	default:
		return s
	}
}
