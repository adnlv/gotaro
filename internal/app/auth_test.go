package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adnlv/gotaro/internal/domain"
	"github.com/adnlv/gotaro/internal/store"
)

type fakeTransactor struct {
	called int
}

func (f *fakeTransactor) WithinTransaction(ctx context.Context, fn func(context.Context, store.Executor) error) error {
	f.called++
	return fn(ctx, nil)
}

type spyUsers struct {
	insertCalls int
	insertErr   error
	last        *domain.User
}

func (s *spyUsers) Insert(ctx context.Context, ex store.Executor, u *domain.User) error {
	s.insertCalls++
	s.last = u
	return s.insertErr
}

func (s *spyUsers) GetByEmail(ctx context.Context, ex store.Executor, email string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (s *spyUsers) GetByID(ctx context.Context, ex store.Executor, id uint64) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

type spySessions struct {
	insertCalls int
	insertErr   error
}

func (s *spySessions) Insert(ctx context.Context, ex store.Executor, sess *domain.Session) error {
	s.insertCalls++
	return s.insertErr
}

func (s *spySessions) DeleteByToken(ctx context.Context, ex store.Executor, token string) error {
	return nil
}

func (s *spySessions) GetValidByToken(ctx context.Context, ex store.Executor, token string) (*domain.Session, error) {
	return nil, domain.ErrUnauthorized
}

type stubHasher struct{}

func (stubHasher) Hash(password string) (string, error) { return "hashed:" + password, nil }

func (stubHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("mismatch")
	}
	return nil
}

func TestRegister_UserInsertFails_NoSessionInsert(t *testing.T) {
	t.Parallel()
	u := &spyUsers{insertErr: errors.New("dup")}
	sess := &spySessions{}
	svc := &AuthService{
		Queries:  nil,
		Tx:       &fakeTransactor{},
		Users:    u,
		Sessions: sess,
		Hasher:   stubHasher{},
	}
	_, err := svc.Register(context.Background(), "a@b.co", "longpassword", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if u.insertCalls != 1 {
		t.Fatalf("user insert calls: %d", u.insertCalls)
	}
	if sess.insertCalls != 0 {
		t.Fatalf("session should not insert, got %d calls", sess.insertCalls)
	}
}

func TestRegister_SessionInsertFails_BothAttempted(t *testing.T) {
	t.Parallel()
	u := &spyUsers{}
	sess := &spySessions{insertErr: errors.New("fail")}
	tx := &fakeTransactor{}
	svc := &AuthService{
		Queries:  nil,
		Tx:       tx,
		Users:    u,
		Sessions: sess,
		Hasher:   stubHasher{},
	}
	_, err := svc.Register(context.Background(), "a@b.co", "longpassword", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if u.insertCalls != 1 || sess.insertCalls != 1 {
		t.Fatalf("users=%d sessions=%d", u.insertCalls, sess.insertCalls)
	}
	if tx.called != 1 {
		t.Fatalf("expected one transaction, got %d", tx.called)
	}
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	u := &spyUsers{}
	sess := &spySessions{}
	svc := &AuthService{
		Queries:  nil,
		Tx:       &fakeTransactor{},
		Users:    u,
		Sessions: sess,
		Hasher:   stubHasher{},
	}
	out, err := svc.Register(context.Background(), "a@b.co", "longpassword", "127.0.0.1", "ua")
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.Token == "" || out.CSRFToken == "" {
		t.Fatalf("session: %+v", out)
	}
	if u.last == nil || u.last.Email != "a@b.co" || u.last.PasswordHash != "hashed:longpassword" {
		t.Fatalf("user: %+v", u.last)
	}
	if !out.ExpiresAt.After(time.Now().UTC()) {
		t.Fatal("expires not in future")
	}
}
