package domain

// TaskStats aggregates per-user task counts for dashboards.
type TaskStats struct {
	Total     int
	Open      int
	Completed int
	Archived  int
	Overdue   int
}
