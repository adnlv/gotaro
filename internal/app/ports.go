package app

import (
	"context"

	"github.com/adnlv/gotaro/internal/domain"
	"github.com/adnlv/gotaro/internal/store"
)

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(context.Context, store.Executor) error) error
}

type UserRepository interface {
	Insert(ctx context.Context, ex store.Executor, u *domain.User) error
	GetByEmail(ctx context.Context, ex store.Executor, email string) (*domain.User, error)
	GetByID(ctx context.Context, ex store.Executor, id uint64) (*domain.User, error)
}

type SessionRepository interface {
	Insert(ctx context.Context, ex store.Executor, s *domain.Session) error
	DeleteByToken(ctx context.Context, ex store.Executor, token string) error
	GetValidByToken(ctx context.Context, ex store.Executor, token string) (*domain.Session, error)
}

type TaskRepository interface {
	Insert(ctx context.Context, ex store.Executor, t *domain.Task) error
	Update(ctx context.Context, ex store.Executor, t *domain.Task) error
	Delete(ctx context.Context, ex store.Executor, userID, taskID uint64) error
	Get(ctx context.Context, ex store.Executor, userID, taskID uint64) (*domain.Task, error)
	List(ctx context.Context, ex store.Executor, q store.TaskQuery) ([]domain.Task, error)
	StatsForUser(ctx context.Context, ex store.Executor, userID uint64) (domain.TaskStats, error)
	ReplaceTaskTags(ctx context.Context, ex store.Executor, userID, taskID uint64, names []string, defaultColor string) error
}
