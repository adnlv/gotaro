package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/adnlv/gotaro/internal/domain"
)

type ProjectRepository struct{}

// Upsert returns the project row for the normalized name, or (nil, nil) if name is empty.
func (ProjectRepository) Upsert(ctx context.Context, ex Executor, userID uint64, name string) (*domain.Project, error) {
	n := domain.NormalizeProjectName(name)
	if n == "" {
		return nil, nil
	}
	const q = `
	INSERT INTO projects (user_id, name) VALUES ($1, $2)
	ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
	RETURNING id, user_id, name
	`
	var p domain.Project
	if err := ex.QueryRowContext(ctx, q, userID, n).Scan(&p.ID, &p.UserID, &p.Name); err != nil {
		return nil, fmt.Errorf("upsert project: %w", err)
	}
	return &p, nil
}

func (ProjectRepository) ListNames(ctx context.Context, ex Executor, userID uint64) ([]string, error) {
	const q = `
	SELECT name
	FROM projects
	WHERE user_id = $1
	ORDER BY name ASC
	`
	rows, err := ex.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
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

func projectFK(p *domain.Project) any {
	if p == nil {
		return nil
	}
	return p.ID
}

func scanProject(pid sql.NullInt64, pname sql.NullString, userID uint64) *domain.Project {
	if !pid.Valid || !pname.Valid {
		return nil
	}
	return &domain.Project{ID: uint64(pid.Int64), UserID: userID, Name: pname.String}
}
