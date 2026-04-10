package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Postgres struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) DB() *sql.DB {
	return p.db
}

// WithinTransaction runs fn with an Executor backed by a single sql.Tx.
// The transaction is committed only if fn returns nil; otherwise it rolls back.
func (p *Postgres) WithinTransaction(ctx context.Context, fn func(context.Context, Executor) error) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
