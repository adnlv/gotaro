package app

import (
	"context"
	"testing"

	"github.com/adnlv/gotaro/internal/domain"
	"github.com/adnlv/gotaro/internal/store"
)

type spyTaskRepo struct {
	listCalls int
}

func (s *spyTaskRepo) Insert(ctx context.Context, ex store.Executor, t *domain.Task) error {
	return nil
}

func (s *spyTaskRepo) Update(ctx context.Context, ex store.Executor, t *domain.Task) error {
	return nil
}

func (s *spyTaskRepo) Delete(ctx context.Context, ex store.Executor, userID, taskID uint64) error {
	return nil
}

func (s *spyTaskRepo) Get(ctx context.Context, ex store.Executor, userID, taskID uint64) (*domain.Task, error) {
	return nil, domain.ErrNotFound
}

func (s *spyTaskRepo) List(ctx context.Context, ex store.Executor, q store.TaskQuery) ([]domain.Task, error) {
	s.listCalls++
	return nil, nil
}

func (s *spyTaskRepo) StatsForUser(ctx context.Context, ex store.Executor, userID uint64) (domain.TaskStats, error) {
	return domain.TaskStats{}, nil
}

func (s *spyTaskRepo) ReplaceTaskTags(ctx context.Context, ex store.Executor, userID, taskID uint64, names []string, defaultColor string) error {
	return nil
}

type noopTransactor struct{}

func (noopTransactor) WithinTransaction(ctx context.Context, fn func(context.Context, store.Executor) error) error {
	return fn(ctx, nil)
}

func TestTaskService_List_InvalidSort_NoListCall(t *testing.T) {
	t.Parallel()
	repo := &spyTaskRepo{}
	svc := &TaskService{
		Queries: nil,
		Tx:      noopTransactor{},
		Tasks:   repo,
	}
	_, err := svc.List(context.Background(), TaskListParams{UserID: 1, SortField: "nope"})
	if err == nil {
		t.Fatal("expected error")
	}
	if repo.listCalls != 0 {
		t.Fatalf("list called %d times", repo.listCalls)
	}
}

func TestTaskService_Create_InvalidTitle_NoTransaction(t *testing.T) {
	t.Parallel()
	tx := &fakeTransactor{}
	repo := &spyTaskRepo{}
	svc := &TaskService{
		Queries: nil,
		Tx:      tx,
		Tasks:   repo,
	}
	_, err := svc.Create(context.Background(), 1, TaskWrite{Title: "  ", Status: domain.StatusTodo, Priority: domain.PriorityMedium})
	if err == nil {
		t.Fatal("expected error")
	}
	if tx.called != 0 {
		t.Fatalf("tx called: %d", tx.called)
	}
}
