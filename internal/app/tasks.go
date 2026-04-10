package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/adnlv/gotaro/internal/domain"
	"github.com/adnlv/gotaro/internal/store"
)

type TaskService struct {
	Queries store.Executor
	Tx      Transactor
	Tasks   TaskRepository
}

type TaskListParams struct {
	UserID uint64

	CompletedView bool
	ArchivedView  bool
	Status          *domain.Status
	Priority        *domain.Priority
	Tag             string
	DueFrom         *time.Time
	DueTo           *time.Time
	Search          string
	SortField       string
	SortDir         string
	Offset          int
	Limit           int
}

func (s *TaskService) List(ctx context.Context, p TaskListParams) ([]domain.Task, error) {
	sortField, err := domain.ValidateSortField(p.SortField)
	if err != nil {
		return nil, err
	}
	sortDir, err := domain.ValidateSortDir(p.SortDir)
	if err != nil {
		return nil, err
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	q := store.TaskQuery{
		UserID:        p.UserID,
		ArchivedOnly:  p.ArchivedView,
		OpenOnly:      !p.ArchivedView && !p.CompletedView && p.Status == nil,
		CompletedOnly: !p.ArchivedView && p.CompletedView,
		Status:        p.Status,
		Priority:      p.Priority,
		Tag:           p.Tag,
		DueFrom:       p.DueFrom,
		DueTo:         p.DueTo,
		Search:        p.Search,
		SortField:     sortField,
		SortDir:       sortDir,
		Offset:        p.Offset,
		Limit:         limit,
	}

	if p.Status != nil {
		q.OpenOnly = false
		q.CompletedOnly = false
	}

	return s.Tasks.List(ctx, s.Queries, q)
}

func (s *TaskService) Stats(ctx context.Context, userID uint64) (domain.TaskStats, error) {
	return s.Tasks.StatsForUser(ctx, s.Queries, userID)
}

func (s *TaskService) Get(ctx context.Context, userID, taskID uint64) (*domain.Task, error) {
	return s.Tasks.Get(ctx, s.Queries, userID, taskID)
}

type TaskWrite struct {
	Title       string
	Description *string
	Status      domain.Status
	Priority    domain.Priority
	DueDate     *time.Time
	TagNames    []string
}

func (s *TaskService) Create(ctx context.Context, userID uint64, w TaskWrite) (*domain.Task, error) {
	if err := domain.ValidateTaskInput(w.Title, w.Description, w.DueDate); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(w.Title)
	now := time.Now().UTC()
	t := &domain.Task{
		UserID:      userID,
		Title:       title,
		Description: w.Description,
		Status:      w.Status,
		Priority:    w.Priority,
		CreatedAt:   now,
		UpdatedAt:   now,
		DueDate:     w.DueDate,
	}

	var created *domain.Task
	err := s.Tx.WithinTransaction(ctx, func(ctx context.Context, ex store.Executor) error {
		if err := s.Tasks.Insert(ctx, ex, t); err != nil {
			return err
		}
		if err := s.Tasks.ReplaceTaskTags(ctx, ex, userID, t.ID, w.TagNames, ""); err != nil {
			return err
		}
		created = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Tasks.Get(ctx, s.Queries, userID, created.ID)
}

func (s *TaskService) Update(ctx context.Context, userID, taskID uint64, w TaskWrite) (*domain.Task, error) {
	if err := domain.ValidateTaskInput(w.Title, w.Description, w.DueDate); err != nil {
		return nil, err
	}
	existing, err := s.Tasks.Get(ctx, s.Queries, userID, taskID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(w.Title)
	now := time.Now().UTC()
	t := &domain.Task{
		ID:          existing.ID,
		UserID:      userID,
		Title:       title,
		Description: w.Description,
		Status:      w.Status,
		Priority:    w.Priority,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   now,
		ArchivedAt:  existing.ArchivedAt,
		DueDate:     w.DueDate,
	}

	err = s.Tx.WithinTransaction(ctx, func(ctx context.Context, ex store.Executor) error {
		if err := s.Tasks.Update(ctx, ex, t); err != nil {
			return err
		}
		return s.Tasks.ReplaceTaskTags(ctx, ex, userID, t.ID, w.TagNames, "")
	})
	if err != nil {
		return nil, err
	}
	return s.Tasks.Get(ctx, s.Queries, userID, taskID)
}

func (s *TaskService) Delete(ctx context.Context, userID, taskID uint64) error {
	err := s.Tasks.Delete(ctx, s.Queries, userID, taskID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrNotFound
	}
	return err
}

func (s *TaskService) SetStatus(ctx context.Context, userID, taskID uint64, st domain.Status) error {
	existing, err := s.Tasks.Get(ctx, s.Queries, userID, taskID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	existing.Status = st
	existing.UpdatedAt = now
	return s.Tasks.Update(ctx, s.Queries, existing)
}

func (s *TaskService) Archive(ctx context.Context, userID, taskID uint64) error {
	existing, err := s.Tasks.Get(ctx, s.Queries, userID, taskID)
	if err != nil {
		return err
	}
	if existing.IsArchived() {
		return nil
	}
	now := time.Now().UTC()
	existing.ArchivedAt = &now
	existing.UpdatedAt = now
	return s.Tasks.Update(ctx, s.Queries, existing)
}

func (s *TaskService) Unarchive(ctx context.Context, userID, taskID uint64) error {
	existing, err := s.Tasks.Get(ctx, s.Queries, userID, taskID)
	if err != nil {
		return err
	}
	if !existing.IsArchived() {
		return nil
	}
	now := time.Now().UTC()
	existing.ArchivedAt = nil
	existing.UpdatedAt = now
	return s.Tasks.Update(ctx, s.Queries, existing)
}
