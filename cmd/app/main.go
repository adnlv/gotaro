package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/adnlv/gotaro/internal/app"
	"github.com/adnlv/gotaro/internal/migrate"
	"github.com/adnlv/gotaro/internal/store"
	"github.com/adnlv/gotaro/internal/web"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
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

	pool := store.NewPostgres(db)
	authSvc := &app.AuthService{
		Queries:  db,
		Tx:       pool,
		Users:    store.UserRepository{},
		Sessions: store.SessionRepository{},
		Hasher:   app.BcryptHasher{},
	}
	taskSvc := &app.TaskService{
		Queries: db,
		Tx:      pool,
		Tasks:   store.TaskRepository{},
	}

	secure := os.Getenv("GOTARO_COOKIE_SECURE") == "true"
	srv, err := web.NewServer(slog.Default(), authSvc, taskSvc, secure)
	if err != nil {
		slog.Error("init server", "err", err)
		os.Exit(1)
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(ctx, addr); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
