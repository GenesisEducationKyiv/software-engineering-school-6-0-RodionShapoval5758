package usecase

import (
	"context"
	"time"

	"GithubReleaseNotificationAPI/services/auth/internal/db"
	"GithubReleaseNotificationAPI/services/auth/internal/user"
)

type loginStore interface {
	GetByEmail(ctx context.Context, q db.DBTX, email string) (user.User, error)
	InsertRefreshToken(ctx context.Context, q db.DBTX, userID, tokenHash string, expiresAt time.Time) error
}

type accessIssuer interface {
	Issue(userID, email string) (string, error)
}

type Login struct {
	store      loginStore
	issuer     accessIssuer
	q          db.DBTX
	refreshTTL time.Duration
}

func NewLogin(store loginStore, issuer accessIssuer, q db.DBTX, refreshTTL time.Duration) *Login {
	return &Login{store: store, issuer: issuer, q: q, refreshTTL: refreshTTL}
}

func (l *Login) Execute(ctx context.Context, email, password string) (TokenPair, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return TokenPair{}, user.ErrInvalidCredentials
	}

	u, err := l.store.GetByEmail(ctx, l.q, email)
	if err != nil {
		return TokenPair{}, err
	}

	ok, err := user.VerifyPassword(password, u.PasswordHash)
	if err != nil || !ok {
		return TokenPair{}, user.ErrInvalidCredentials
	}

	if !u.EmailVerified {
		return TokenPair{}, user.ErrEmailNotVerified
	}

	accessToken, err := l.issuer.Issue(u.ID, u.Email)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := newRawToken()
	if err != nil {
		return TokenPair{}, err
	}

	expiresAt := time.Now().Add(l.refreshTTL)
	if err := l.store.InsertRefreshToken(ctx, l.q, u.ID, hashToken(refreshToken), expiresAt); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
