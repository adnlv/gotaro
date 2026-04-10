package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/adnlv/gotaro/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository struct{}

func (UserRepository) Insert(ctx context.Context, ex Executor, u *domain.User) error {
	const q = `
	INSERT INTO users (email, password_hash, registered_at, updated_at)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`
	if err := ex.QueryRowContext(ctx, q, u.Email, u.PasswordHash, u.RegisteredAt, u.UpdatedAt).Scan(&u.ID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateEmail
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (UserRepository) GetByEmail(ctx context.Context, ex Executor, email string) (*domain.User, error) {
	const q = `
	SELECT id, email, password_hash, registered_at, updated_at
	FROM users
	WHERE email = $1
	`
	var u domain.User
	if err := ex.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.RegisteredAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (UserRepository) GetByID(ctx context.Context, ex Executor, id uint64) (*domain.User, error) {
	const q = `
	SELECT id, email, password_hash, registered_at, updated_at
	FROM users
	WHERE id = $1
	`
	var u domain.User
	if err := ex.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.RegisteredAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}
