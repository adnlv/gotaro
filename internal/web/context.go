package web

import (
	"context"

	"github.com/adnlv/gotaro/internal/domain"
)

type ctxKey int

const sessionKey ctxKey = 1

func withSession(ctx context.Context, s *domain.Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

func sessionFrom(ctx context.Context) (*domain.Session, bool) {
	v := ctx.Value(sessionKey)
	if v == nil {
		return nil, false
	}
	s, ok := v.(*domain.Session)
	return s, ok
}
