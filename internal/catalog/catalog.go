package catalog

import (
	"GithubReleaseNotificationAPI/internal/catalog/internal/store"
	"GithubReleaseNotificationAPI/internal/catalog/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewEnsure(pool *pgxpool.Pool) *usecase.Ensure {
	return usecase.NewEnsure(store.New(pool))
}

func NewDeleteIfOrphaned(pool *pgxpool.Pool) *usecase.DeleteIfOrphaned {
	return usecase.NewDeleteIfOrphaned(store.New(pool))
}

func NewListTracked(pool *pgxpool.Pool) *usecase.ListTracked {
	return usecase.NewListTracked(store.New(pool))
}

func NewUpdateLastSeenTagAtomic(pool *pgxpool.Pool) *usecase.UpdateLastSeenTagAtomic {
	return usecase.NewUpdateLastSeenTagAtomic(pool)
}
