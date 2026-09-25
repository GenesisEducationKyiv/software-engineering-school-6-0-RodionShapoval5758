package usecase

import (
	"context"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/services/auth/internal/db"
	"GithubReleaseNotificationAPI/services/auth/internal/user"

	"github.com/jackc/pgx/v5"
)

type refreshStore interface {
	GetRefreshTokenForUpdate(ctx context.Context, q db.DBTX, tokenHash string) (user.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, q db.DBTX, tokenHash string) error
	InsertRefreshToken(ctx context.Context, q db.DBTX, userID, tokenHash string, expiresAt time.Time) error
}

type Refresh struct {
	store      refreshStore
	issuer     accessIssuer
	txer       db.TxBeginner
	refreshTTL time.Duration
}

func NewRefresh(store refreshStore, issuer accessIssuer, txer db.TxBeginner, refreshTTL time.Duration) *Refresh {
	return &Refresh{store: store, issuer: issuer, txer: txer, refreshTTL: refreshTTL}
}

func (r *Refresh) Execute(ctx context.Context, rawToken string) (TokenPair, error) {
	oldHash := hashToken(rawToken)

	tx, err := r.txer.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TokenPair{}, fmt.Errorf("begin refresh tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stored, err := r.store.GetRefreshTokenForUpdate(ctx, tx, oldHash)
	if err != nil {
		return TokenPair{}, err
	}

	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return TokenPair{}, user.ErrRefreshTokenInvalid
	}

	if err := r.store.RevokeRefreshToken(ctx, tx, oldHash); err != nil {
		return TokenPair{}, err
	}

	newToken, err := newRawToken()
	if err != nil {
		return TokenPair{}, err
	}

	expiresAt := time.Now().Add(r.refreshTTL)
	if err := r.store.InsertRefreshToken(ctx, tx, stored.UserID, hashToken(newToken), expiresAt); err != nil {
		return TokenPair{}, err
	}

	accessToken, err := r.issuer.Issue(stored.UserID, stored.Email)
	if err != nil {
		return TokenPair{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return TokenPair{}, fmt.Errorf("commit refresh tx: %w", err)
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: newToken}, nil
}
