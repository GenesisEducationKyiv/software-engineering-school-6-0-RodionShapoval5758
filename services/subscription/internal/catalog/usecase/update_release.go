package usecase

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/services/subscription/internal/catalog/internal/store"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UpdateLastSeenTagAtomic struct {
	pool *pgxpool.Pool
}

func NewUpdateLastSeenTagAtomic(pool *pgxpool.Pool) *UpdateLastSeenTagAtomic {
	return &UpdateLastSeenTagAtomic{pool: pool}
}

func (uc *UpdateLastSeenTagAtomic) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error {
	tx, err := uc.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txStore := store.New(tx)
	if err := txStore.UpdateLastSeenTag(ctx, repoID, tag); err != nil {
		return err
	}

	if err := onTx(ctx, tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
