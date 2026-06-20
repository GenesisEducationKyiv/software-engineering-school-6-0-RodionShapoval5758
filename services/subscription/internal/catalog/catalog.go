package catalog

import (
	"context"

	"GithubReleaseNotificationAPI/services/subscription/internal/catalog/internal/store"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog/usecase"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DeleteOrphanedOutbox is satisfied by *outbox.Store. Defined here so callers
// don't need to import internal/catalog/usecase directly.
type DeleteOrphanedOutbox interface {
	Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error
}

func NewEnsure(pool *pgxpool.Pool) *usecase.Ensure {
	return usecase.NewEnsure(store.New(pool))
}

func NewDeleteIfOrphaned(pool *pgxpool.Pool, outbox DeleteOrphanedOutbox) *usecase.DeleteIfOrphaned {
	return usecase.NewDeleteIfOrphaned(store.New(pool), pool, outbox)
}

func NewListTracked(pool *pgxpool.Pool) *usecase.ListTracked {
	return usecase.NewListTracked(store.New(pool))
}

func NewUpdateLastSeenTagAtomic(pool *pgxpool.Pool) *usecase.UpdateLastSeenTagAtomic {
	return usecase.NewUpdateLastSeenTagAtomic(pool)
}
