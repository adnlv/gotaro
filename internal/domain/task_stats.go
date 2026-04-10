package domain

import "time"

// TaskStats aggregates per-user task counts for dashboards.
type TaskStats struct {
	Total     int
	Open      int
	Completed int
	Archived  int
	Overdue   int
}

// DailyActivityPoint is one calendar day (UTC) of task flow metrics.
// Completed uses tasks that are currently done, bucketed by updated_at (UTC date)—a simple proxy for “finished that day”.
type DailyActivityPoint struct {
	Date      time.Time
	Created   int
	Completed int
}
