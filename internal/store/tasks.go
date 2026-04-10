package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adnlv/gotaro/internal/domain"
)

type TaskQuery struct {
	UserID uint64

	OpenOnly      bool
	CompletedOnly bool
	ArchivedOnly  bool

	Status   *domain.Status
	Priority *domain.Priority
	Tag      string
	DueFrom  *time.Time
	DueTo    *time.Time
	Search   string
	Project  string

	SortField string
	SortDir   string

	Limit  int
	Offset int
}

type TaskRepository struct{}

func (TaskRepository) Insert(ctx context.Context, ex Executor, t *domain.Task) error {
	const q = `
	INSERT INTO tasks (user_id, title, description, status, priority, created_at, updated_at, archived_at, due_date, project_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id
	`
	if err := ex.QueryRowContext(ctx, q,
		t.UserID,
		t.Title,
		nullableString(t.Description),
		int16(t.Status),
		int16(t.Priority),
		t.CreatedAt,
		t.UpdatedAt,
		nullableTime(t.ArchivedAt),
		nullableTime(t.DueDate),
		projectFK(t.Project),
	).Scan(&t.ID); err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (TaskRepository) Update(ctx context.Context, ex Executor, t *domain.Task) error {
	const q = `
	UPDATE tasks
	SET title = $1, description = $2, status = $3, priority = $4, updated_at = $5, archived_at = $6, due_date = $7, project_id = $8
	WHERE id = $9 AND user_id = $10
	`
	res, err := ex.ExecContext(ctx, q,
		t.Title,
		nullableString(t.Description),
		int16(t.Status),
		int16(t.Priority),
		t.UpdatedAt,
		nullableTime(t.ArchivedAt),
		nullableTime(t.DueDate),
		projectFK(t.Project),
		t.ID,
		t.UserID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (TaskRepository) Delete(ctx context.Context, ex Executor, userID, taskID uint64) error {
	const q = `DELETE FROM tasks WHERE id = $1 AND user_id = $2`
	res, err := ex.ExecContext(ctx, q, taskID, userID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (TaskRepository) Get(ctx context.Context, ex Executor, userID, taskID uint64) (*domain.Task, error) {
	const q = `
	SELECT t.id, t.user_id, t.title, t.description, t.status, t.priority, t.created_at, t.updated_at, t.archived_at, t.due_date,
	       p.id, p.name
	FROM tasks t
	LEFT JOIN projects p ON p.id = t.project_id AND p.user_id = t.user_id
	WHERE t.id = $1 AND t.user_id = $2
	`
	var t domain.Task
	var desc sql.NullString
	var archived, due sql.NullTime
	var pid sql.NullInt64
	var pname sql.NullString
	if err := ex.QueryRowContext(ctx, q, taskID, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &desc, &t.Status, &t.Priority,
		&t.CreatedAt, &t.UpdatedAt, &archived, &due,
		&pid, &pname,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	t.Description = nullStringToPtr(desc)
	t.ArchivedAt = nullTimeToPtr(archived)
	t.DueDate = nullTimeToPtr(due)
	t.Project = scanProject(pid, pname, userID)

	tags, err := loadTagsForTask(ctx, ex, t.ID)
	if err != nil {
		return nil, err
	}
	t.Tags = tags
	return &t, nil
}

// StatsForUser returns aggregate counts for the user's tasks (matches domain overdue rule in UTC).
func (TaskRepository) StatsForUser(ctx context.Context, ex Executor, userID uint64) (domain.TaskStats, error) {
	done := int16(domain.StatusDone)
	const q = `
	SELECT
		COUNT(*)::int,
		COUNT(*) FILTER (WHERE archived_at IS NULL AND status <> $2)::int,
		COUNT(*) FILTER (WHERE status = $2)::int,
		COUNT(*) FILTER (WHERE archived_at IS NOT NULL)::int,
		COUNT(*) FILTER (
			WHERE archived_at IS NULL
			AND status <> $2
			AND due_date IS NOT NULL
			AND (due_date AT TIME ZONE 'UTC')::date < (timezone('UTC', now()))::date
		)::int
	FROM tasks
	WHERE user_id = $1
	`
	var s domain.TaskStats
	if err := ex.QueryRowContext(ctx, q, userID, done).Scan(
		&s.Total, &s.Open, &s.Completed, &s.Archived, &s.Overdue,
	); err != nil {
		return domain.TaskStats{}, fmt.Errorf("task stats: %w", err)
	}
	return s, nil
}

// DailyActivity returns one row per UTC calendar day from start..end inclusive (last `days` days through today).
func (TaskRepository) DailyActivity(ctx context.Context, ex Executor, userID uint64, days int) ([]domain.DailyActivityPoint, error) {
	if days < 1 {
		days = 14
	}
	if days > 90 {
		days = 90
	}
	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -(days - 1))

	created := make(map[string]int)
	const qCreated = `
	SELECT (created_at AT TIME ZONE 'UTC')::date, COUNT(*)::int
	FROM tasks
	WHERE user_id = $1
	  AND (created_at AT TIME ZONE 'UTC')::date >= $2::date
	  AND (created_at AT TIME ZONE 'UTC')::date <= $3::date
	GROUP BY 1
	`
	rows, err := ex.QueryContext(ctx, qCreated, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("daily created: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var d time.Time
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			return nil, err
		}
		created[d.Format("2006-01-02")] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	completed := make(map[string]int)
	done := int16(domain.StatusDone)
	const qDone = `
	SELECT (updated_at AT TIME ZONE 'UTC')::date, COUNT(*)::int
	FROM tasks
	WHERE user_id = $1 AND status = $2
	  AND (updated_at AT TIME ZONE 'UTC')::date >= $3::date
	  AND (updated_at AT TIME ZONE 'UTC')::date <= $4::date
	GROUP BY 1
	`
	rows2, err := ex.QueryContext(ctx, qDone, userID, done, start, end)
	if err != nil {
		return nil, fmt.Errorf("daily completed: %w", err)
	}
	defer func() { _ = rows2.Close() }()

	for rows2.Next() {
		var d time.Time
		var n int
		if err := rows2.Scan(&d, &n); err != nil {
			return nil, err
		}
		completed[d.Format("2006-01-02")] = n
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	out := make([]domain.DailyActivityPoint, 0, days)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		out = append(out, domain.DailyActivityPoint{
			Date:      d,
			Created:   created[k],
			Completed: completed[k],
		})
	}
	return out, nil
}

func (TaskRepository) ListTagNames(ctx context.Context, ex Executor, userID uint64) ([]string, error) {
	const q = `
	SELECT name
	FROM tags
	WHERE user_id = $1
	ORDER BY name ASC
	`
	rows, err := ex.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list tag names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

func (TaskRepository) List(ctx context.Context, ex Executor, q TaskQuery) ([]domain.Task, error) {
	if q.Limit <= 0 {
		q.Limit = 100
	}
	if q.Limit > 500 {
		q.Limit = 500
	}

	var b strings.Builder
	b.WriteString(`
	SELECT DISTINCT t.id, t.user_id, t.title, t.description, t.status, t.priority,
	       t.created_at, t.updated_at, t.archived_at, t.due_date,
	       p.id, p.name
	FROM tasks t
	LEFT JOIN projects p ON p.id = t.project_id AND p.user_id = t.user_id
	`)
	args := []any{}
	arg := 1

	if strings.TrimSpace(q.Tag) != "" {
		b.WriteString(`
		INNER JOIN task_tags tt ON tt.task_id = t.id
		INNER JOIN tags tg ON tg.id = tt.tag_id AND tg.user_id = t.user_id
		`)
	}

	b.WriteString(` WHERE t.user_id = $`)
	b.WriteString(fmt.Sprint(arg))
	args = append(args, q.UserID)
	arg++

	if q.ArchivedOnly {
		b.WriteString(` AND t.archived_at IS NOT NULL`)
	} else {
		b.WriteString(` AND t.archived_at IS NULL`)
	}

	if q.CompletedOnly {
		b.WriteString(` AND t.status = $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, int16(domain.StatusDone))
		arg++
	} else if q.OpenOnly {
		b.WriteString(` AND t.status <> $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, int16(domain.StatusDone))
		arg++
	}

	if q.Status != nil {
		b.WriteString(` AND t.status = $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, int16(*q.Status))
		arg++
	}
	if q.Priority != nil {
		b.WriteString(` AND t.priority = $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, int16(*q.Priority))
		arg++
	}
	if tag := strings.TrimSpace(q.Tag); tag != "" {
		b.WriteString(` AND tg.name = $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, domain.NormalizeTagName(tag))
		arg++
	}
	if proj := strings.TrimSpace(q.Project); proj != "" {
		b.WriteString(` AND p.name = $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, domain.NormalizeProjectName(proj))
		arg++
	}
	if q.DueFrom != nil {
		b.WriteString(` AND t.due_date IS NOT NULL AND t.due_date >= $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, *q.DueFrom)
		arg++
	}
	if q.DueTo != nil {
		b.WriteString(` AND t.due_date IS NOT NULL AND t.due_date <= $`)
		b.WriteString(fmt.Sprint(arg))
		args = append(args, *q.DueTo)
		arg++
	}
	if s := strings.TrimSpace(q.Search); s != "" {
		pat := "%" + strings.ReplaceAll(s, "%", `\%`) + "%"
		b.WriteString(` AND (t.title ILIKE $`)
		b.WriteString(fmt.Sprint(arg))
		b.WriteString(` OR t.description ILIKE $`)
		b.WriteString(fmt.Sprint(arg + 1))
		b.WriteString(`)`)
		args = append(args, pat, pat)
		arg += 2
	}

	orderCol := "t.created_at"
	nullsLast := false
	switch q.SortField {
	case "due_date":
		orderCol = "t.due_date"
		nullsLast = true
	case "priority":
		orderCol = "t.priority"
	case "status":
		orderCol = "t.status"
	case "created_at", "":
		orderCol = "t.created_at"
	}
	dir := "DESC"
	if q.SortDir == "ASC" {
		dir = "ASC"
	}
	b.WriteString(` ORDER BY `)
	b.WriteString(orderCol)
	b.WriteString(` `)
	b.WriteString(dir)
	if nullsLast {
		b.WriteString(` NULLS LAST`)
	}
	b.WriteString(`, t.id `)
	b.WriteString(dir)

	b.WriteString(` LIMIT $`)
	b.WriteString(fmt.Sprint(arg))
	args = append(args, q.Limit)
	arg++

	b.WriteString(` OFFSET $`)
	b.WriteString(fmt.Sprint(arg))
	args = append(args, q.Offset)

	rows, err := ex.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	var ids []uint64
	for rows.Next() {
		var t domain.Task
		var desc sql.NullString
		var archived, due sql.NullTime
		var pid sql.NullInt64
		var pname sql.NullString
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &desc, &t.Status, &t.Priority,
			&t.CreatedAt, &t.UpdatedAt, &archived, &due,
			&pid, &pname,
		); err != nil {
			return nil, err
		}
		t.Description = nullStringToPtr(desc)
		t.ArchivedAt = nullTimeToPtr(archived)
		t.DueDate = nullTimeToPtr(due)
		t.Project = scanProject(pid, pname, q.UserID)
		t.Tags = nil
		tasks = append(tasks, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return tasks, nil
	}
	tagMap, err := loadTagsForTasks(ctx, ex, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].Tags = tagMap[tasks[i].ID]
	}
	return tasks, nil
}

func (TaskRepository) ReplaceTaskTags(ctx context.Context, ex Executor, userID, taskID uint64, names []string, defaultColor string) error {
	if defaultColor == "" {
		defaultColor = "#6c757d"
	}
	const del = `DELETE FROM task_tags WHERE task_id = $1`
	if _, err := ex.ExecContext(ctx, del, taskID); err != nil {
		return fmt.Errorf("clear task tags: %w", err)
	}
	for _, name := range names {
		n := domain.NormalizeTagName(name)
		if n == "" {
			continue
		}
		tagID, err := upsertTag(ctx, ex, userID, n, defaultColor)
		if err != nil {
			return err
		}
		const ins = `INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
		if _, err := ex.ExecContext(ctx, ins, taskID, tagID); err != nil {
			return fmt.Errorf("link tag: %w", err)
		}
	}
	return nil
}

func upsertTag(ctx context.Context, ex Executor, userID uint64, name, color string) (uint64, error) {
	const ins = `
	INSERT INTO tags (user_id, name, color) VALUES ($1, $2, $3)
	ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
	RETURNING id
	`
	var id uint64
	if err := ex.QueryRowContext(ctx, ins, userID, name, color).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert tag: %w", err)
	}
	return id, nil
}

func loadTagsForTask(ctx context.Context, ex Executor, taskID uint64) ([]domain.Tag, error) {
	m, err := loadTagsForTasks(ctx, ex, []uint64{taskID})
	if err != nil {
		return nil, err
	}
	return m[taskID], nil
}

func loadTagsForTasks(ctx context.Context, ex Executor, taskIDs []uint64) (map[uint64][]domain.Tag, error) {
	out := make(map[uint64][]domain.Tag)
	if len(taskIDs) == 0 {
		return out, nil
	}
	ph := make([]string, len(taskIDs))
	args := make([]any, len(taskIDs))
	for i, id := range taskIDs {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	q := fmt.Sprintf(`
	SELECT tt.task_id, tg.id, tg.user_id, tg.name, tg.color
	FROM task_tags tt
	JOIN tags tg ON tg.id = tt.tag_id
	WHERE tt.task_id IN (%s)
	ORDER BY tg.name
	`, strings.Join(ph, ","))
	rows, err := ex.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var taskID, id, userID uint64
		var name, color string
		if err := rows.Scan(&taskID, &id, &userID, &name, &color); err != nil {
			return nil, err
		}
		out[taskID] = append(out[taskID], domain.Tag{ID: id, UserID: userID, Name: name, Color: color})
	}
	return out, rows.Err()
}

func nullableString(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func nullableTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return *t
}

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}
