package usecase

import (
	"context"

	"GithubReleaseNotificationAPI/services/auth/internal/db"
)

type logoutStore interface {
	RevokeRefreshToken(ctx context.Context, q db.DBTX, tokenHash string) error
}

type Logout struct {
	store logoutStore
	q     db.DBTX
}

func NewLogout(store logoutStore, q db.DBTX) *Logout {
	return &Logout{store: store, q: q}
}

func (l *Logout) Execute(ctx context.Context, rawToken string) error {
	return l.store.RevokeRefreshToken(ctx, l.q, hashToken(rawToken))
}
