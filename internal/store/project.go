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
