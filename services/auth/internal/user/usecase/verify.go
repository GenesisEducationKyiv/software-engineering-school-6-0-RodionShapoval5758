package usecase

import (
	"context"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/services/auth/internal/db"
	"GithubReleaseNotificationAPI/services/auth/internal/user"

	"github.com/jackc/pgx/v5"
)

type verifyStore interface {
	GetVerificationForUpdate(ctx context.Context, q db.DBTX, token string) (user.Verification, error)
	MarkVerificationUsed(ctx context.Context, q db.DBTX, token string) error
	MarkUserVerified(ctx context.Context, q db.DBTX, userID string) error
}

type VerifyEmail struct {
	store verifyStore
	txer  db.TxBeginner
}

func NewVerifyEmail(store verifyStore, txer db.TxBeginner) *VerifyEmail {
	return &VerifyEmail{store: store, txer: txer}
}

func (v *VerifyEmail) Execute(ctx context.Context, token string) error {
	tx, err := v.txer.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin verify tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	verification, err := v.store.GetVerificationForUpdate(ctx, tx, token)
	if err != nil {
		return err
	}

	if verification.UsedAt != nil {
		return user.ErrTokenUsed
	}

	if time.Now().After(verification.ExpiresAt) {
		return user.ErrTokenExpired
	}

	if err := v.store.MarkVerificationUsed(ctx, tx, token); err != nil {
		return err
	}

	if err := v.store.MarkUserVerified(ctx, tx, verification.UserID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
