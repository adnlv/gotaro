package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/adnlv/gotaro/internal/domain"
)

type SessionRepository struct{}

func (SessionRepository) Insert(ctx context.Context, ex Executor, s *domain.Session) error {
	const q = `
	INSERT INTO sessions (token, user_id, csrf_token, client_ip, user_agent, created_at, updated_at, expires_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := ex.ExecContext(ctx, q,
		s.Token,
		s.User.ID,
		s.CSRFToken,
		s.ClientIP,
		s.UserAgent,
		s.CreatedAt,
		s.UpdatedAt,
		s.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (SessionRepository) DeleteByToken(ctx context.Context, ex Executor, token string) error {
	const q = `DELETE FROM sessions WHERE token = $1`
	_, err := ex.ExecContext(ctx, q, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (SessionRepository) GetValidByToken(ctx context.Context, ex Executor, token string) (*domain.Session, error) {
	const q = `
	SELECT
		s.token, s.csrf_token, s.client_ip, s.user_agent,
		s.created_at, s.updated_at, s.expires_at,
		u.id, u.email, u.registered_at, u.updated_at
	FROM sessions s
	JOIN users u ON u.id = s.user_id
	WHERE s.token = $1 AND s.expires_at > $2
	`
	var s domain.Session
	var u domain.User
	now := time.Now().UTC()
	if err := ex.QueryRowContext(ctx, q, token, now).Scan(
		&s.Token, &s.CSRFToken, &s.ClientIP, &s.UserAgent,
		&s.CreatedAt, &s.UpdatedAt, &s.ExpiresAt,
		&u.ID, &u.Email, &u.RegisteredAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUnauthorized
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	s.User = &u
	return &s, nil
}
