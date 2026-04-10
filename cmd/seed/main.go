package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/adnlv/gotaro/internal/migrate"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type cfg struct {
	email       string
	password    string
	taskCount   int
	projectPool int
	tagPool     int
	wipeData    bool
	seed        int64
}

func main() {
	c := cfg{}
	flag.StringVar(&c.email, "email", "test@example.com", "email of the seeded user")
	flag.StringVar(&c.password, "password", "test12345", "password for the seeded user")
	flag.IntVar(&c.taskCount, "tasks", 80, "number of tasks to create")
	flag.IntVar(&c.projectPool, "projects", 6, "how many distinct projects to seed")
	flag.IntVar(&c.tagPool, "tags", 10, "how many distinct tags to seed")
	flag.BoolVar(&c.wipeData, "wipe", true, "delete the user's existing tasks/projects/tags before seeding")
	flag.Int64Var(&c.seed, "seed", 0, "random seed (0 = current unix time)")
	flag.Parse()

	if strings.TrimSpace(c.email) == "" {
		slog.Error("email is required")
		os.Exit(1)
	}
	if len(c.password) < 8 {
		slog.Error("password must be at least 8 characters")
		os.Exit(1)
	}
	if c.taskCount < 1 {
		slog.Error("tasks must be >= 1")
		os.Exit(1)
	}
	if c.projectPool < 0 || c.tagPool < 0 {
		slog.Error("projects/tags must be >= 0")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("ping database", "err", err)
		os.Exit(1)
	}
	if err := migrate.Up(db); err != nil {
		slog.Error("run migrations", "err", err)
		os.Exit(1)
	}

	if c.seed == 0 {
		c.seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(c.seed))

	ctx := context.Background()
	userID, err := upsertUser(ctx, db, c.email, c.password)
	if err != nil {
		slog.Error("upsert user", "err", err)
		os.Exit(1)
	}

	if c.wipeData {
		if err := wipeUserData(ctx, db, userID); err != nil {
			slog.Error("wipe existing user data", "err", err)
			os.Exit(1)
		}
	}

	projects, err := upsertProjects(ctx, db, userID, c.projectPool)
	if err != nil {
		slog.Error("seed projects", "err", err)
		os.Exit(1)
	}
	tags, err := upsertTags(ctx, db, userID, c.tagPool)
	if err != nil {
		slog.Error("seed tags", "err", err)
		os.Exit(1)
	}

	if err := seedTasks(ctx, db, rng, userID, c.taskCount, projects, tags); err != nil {
		slog.Error("seed tasks", "err", err)
		os.Exit(1)
	}

	fmt.Printf("Seed complete.\n")
	fmt.Printf("User: %s (id=%d)\n", c.email, userID)
	fmt.Printf("Password: %s\n", c.password)
	fmt.Printf("Tasks created: %d\n", c.taskCount)
	fmt.Printf("Projects available: %d\n", len(projects))
	fmt.Printf("Tags available: %d\n", len(tags))
	fmt.Printf("Random seed: %d\n", c.seed)
}

type projectRow struct {
	id   int64
	name string
}

type tagRow struct {
	id    int64
	name  string
	color string
}

func upsertUser(ctx context.Context, db *sql.DB, email, password string) (int64, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("bcrypt hash: %w", err)
	}

	const q = `
	INSERT INTO users (email, password_hash, registered_at, updated_at)
	VALUES ($1, $2, now(), now())
	ON CONFLICT (email) DO UPDATE
	  SET password_hash = EXCLUDED.password_hash,
	      updated_at = EXCLUDED.updated_at
	RETURNING id
	`
	var id int64
	if err := db.QueryRowContext(ctx, q, strings.TrimSpace(strings.ToLower(email)), string(hashBytes)).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert users: %w", err)
	}
	return id, nil
}

func wipeUserData(ctx context.Context, db *sql.DB, userID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete projects: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete tags: %w", err)
	}
	return tx.Commit()
}

func upsertProjects(ctx context.Context, db *sql.DB, userID int64, count int) ([]projectRow, error) {
	base := []string{"Inbox", "Work", "Personal", "Home", "Health", "Learning", "Finance", "Errands", "Planning", "Team"}
	if count > len(base) {
		count = len(base)
	}
	if count <= 0 {
		return nil, nil
	}

	rows := make([]projectRow, 0, count)
	for i := 0; i < count; i++ {
		const q = `
		INSERT INTO projects (user_id, name)
		VALUES ($1, $2)
		ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name
		`
		var p projectRow
		if err := db.QueryRowContext(ctx, q, userID, base[i]).Scan(&p.id, &p.name); err != nil {
			return nil, fmt.Errorf("upsert project %q: %w", base[i], err)
		}
		rows = append(rows, p)
	}
	return rows, nil
}

func upsertTags(ctx context.Context, db *sql.DB, userID int64, count int) ([]tagRow, error) {
	base := []tagRow{
		{name: "backend", color: "#3b82f6"},
		{name: "frontend", color: "#06b6d4"},
		{name: "ops", color: "#64748b"},
		{name: "bug", color: "#ef4444"},
		{name: "feature", color: "#22c55e"},
		{name: "urgent", color: "#f59e0b"},
		{name: "docs", color: "#7c3aed"},
		{name: "refactor", color: "#0ea5e9"},
		{name: "research", color: "#14b8a6"},
		{name: "chore", color: "#94a3b8"},
		{name: "qa", color: "#10b981"},
		{name: "customer", color: "#ec4899"},
	}
	if count > len(base) {
		count = len(base)
	}
	if count <= 0 {
		return nil, nil
	}

	out := make([]tagRow, 0, count)
	for i := 0; i < count; i++ {
		const q = `
		INSERT INTO tags (user_id, name, color)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, name) DO UPDATE SET color = EXCLUDED.color
		RETURNING id, name, color
		`
		var t tagRow
		if err := db.QueryRowContext(ctx, q, userID, base[i].name, base[i].color).Scan(&t.id, &t.name, &t.color); err != nil {
			return nil, fmt.Errorf("upsert tag %q: %w", base[i].name, err)
		}
		out = append(out, t)
	}
	return out, nil
}

func seedTasks(ctx context.Context, db *sql.DB, rng *rand.Rand, userID int64, taskCount int, projects []projectRow, tags []tagRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	titlePrefixes := []string{
		"Implement", "Review", "Fix", "Draft", "Plan", "Refactor", "Test", "Document",
		"Investigate", "Prepare", "Improve", "Polish", "Ship", "Validate", "Design",
	}
	titleTargets := []string{
		"login flow", "task filters", "CSV export", "dashboard stats", "project assignment",
		"tagging UX", "archiving behavior", "overdue badge", "error handling", "API contract",
		"seed tooling", "build pipeline", "deployment checklist", "session handling", "permissions",
	}
	descriptionBits := []string{
		"Include edge cases and test coverage.",
		"Confirm behavior in open/completed/archived views.",
		"Coordinate with stakeholders before release.",
		"Keep implementation simple and maintainable.",
		"Document assumptions in the PR.",
	}

	const insertTask = `
	INSERT INTO tasks (
		user_id, title, description, status, priority,
		created_at, updated_at, archived_at, due_date, project_id
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id
	`
	const insertTaskTag = `
	INSERT INTO task_tags (task_id, tag_id)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	`

	now := time.Now().UTC()
	for i := 0; i < taskCount; i++ {
		status := weightedStatus(rng)
		priority := weightedPriority(rng)

		createdAt := now.Add(-time.Duration(rng.Intn(120*24)) * time.Hour)
		updatedAt := createdAt.Add(time.Duration(rng.Intn(96)+1) * time.Hour)
		if updatedAt.After(now) {
			updatedAt = now.Add(-time.Duration(rng.Intn(24)) * time.Hour)
		}

		var archivedAt any
		if rng.Float64() < 0.2 {
			a := updatedAt.Add(time.Duration(rng.Intn(72)+1) * time.Hour)
			if a.After(now) {
				a = now
			}
			archivedAt = a
		}

		var dueDate any
		if rng.Float64() < 0.75 {
			dayOffset := rng.Intn(61) - 20 // -20..+40 days
			d := now.AddDate(0, 0, dayOffset)
			dueDate = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
		}

		var projectID any
		if len(projects) > 0 && rng.Float64() < 0.8 {
			projectID = projects[rng.Intn(len(projects))].id
		}

		title := fmt.Sprintf("%s %s", titlePrefixes[rng.Intn(len(titlePrefixes))], titleTargets[rng.Intn(len(titleTargets))])
		var description any
		if rng.Float64() < 0.8 {
			description = fmt.Sprintf("%s %s", strings.Title(title), descriptionBits[rng.Intn(len(descriptionBits))])
		}

		var taskID int64
		if err := tx.QueryRowContext(ctx, insertTask,
			userID, title, description, status, priority,
			createdAt, updatedAt, archivedAt, dueDate, projectID,
		).Scan(&taskID); err != nil {
			return fmt.Errorf("insert task %d: %w", i+1, err)
		}

		if len(tags) > 0 {
			tagCount := rng.Intn(4) // 0..3
			if tagCount > len(tags) {
				tagCount = len(tags)
			}
			perm := rng.Perm(len(tags))
			for j := 0; j < tagCount; j++ {
				if _, err := tx.ExecContext(ctx, insertTaskTag, taskID, tags[perm[j]].id); err != nil {
					return fmt.Errorf("attach tag to task %d: %w", taskID, err)
				}
			}
		}
	}

	return tx.Commit()
}

func weightedStatus(rng *rand.Rand) int16 {
	// domain.Status: todo=0, in_progress=1, done=2
	p := rng.Float64()
	switch {
	case p < 0.42:
		return 0
	case p < 0.74:
		return 1
	default:
		return 2
	}
}

func weightedPriority(rng *rand.Rand) int16 {
	// domain.Priority: low=-1, medium=0, high=1
	p := rng.Float64()
	switch {
	case p < 0.25:
		return -1
	case p < 0.78:
		return 0
	default:
		return 1
	}
}
