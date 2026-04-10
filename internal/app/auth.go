package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adnlv/gotaro/internal/domain"
	"github.com/adnlv/gotaro/internal/store"
	"github.com/google/uuid"
)

const sessionTTL = 30 * 24 * time.Hour

type AuthService struct {
	Queries  store.Executor
	Tx       Transactor
	Users    UserRepository
	Sessions SessionRepository
	Hasher   PasswordHasher
}

// Register creates a user and a session in a single transaction.
func (s *AuthService) Register(ctx context.Context, email, password, clientIP, userAgent string) (*domain.Session, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if err := domain.ValidateCredentials(email, password); err != nil {
		return nil, err
	}
	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		RegisteredAt: now,
		UpdatedAt:    now,
	}

	var out *domain.Session
	err = s.Tx.WithinTransaction(ctx, func(ctx context.Context, ex store.Executor) error {
		if err := s.Users.Insert(ctx, ex, user); err != nil {
			return err
		}
		sess, err := newSession(user, clientIP, userAgent, now)
		if err != nil {
			return err
		}
		if err := s.Sessions.Insert(ctx, ex, sess); err != nil {
			return err
		}
		out = sess
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AuthService) Login(ctx context.Context, email, password, clientIP, userAgent string) (*domain.Session, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if err := domain.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return nil, err
	}

	user, err := s.Users.GetByEmail(ctx, s.Queries, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if err := s.Hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, domain.ErrUnauthorized
	}

	now := time.Now().UTC()
	sess, err := newSession(user, clientIP, userAgent, now)
	if err != nil {
		return nil, err
	}
	if err := s.Sessions.Insert(ctx, s.Queries, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.Sessions.DeleteByToken(ctx, s.Queries, token)
}

func (s *AuthService) Session(ctx context.Context, token string) (*domain.Session, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.Sessions.GetValidByToken(ctx, s.Queries, token)
}

func newSession(user *domain.User, clientIP, userAgent string, now time.Time) (*domain.Session, error) {
	tok, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("session token: %w", err)
	}
	csrf, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("csrf token: %w", err)
	}
	return &domain.Session{
		Token:     tok.String(),
		CSRFToken: csrf.String(),
		User:      user,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}, nil
}
