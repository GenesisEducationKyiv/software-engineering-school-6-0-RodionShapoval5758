package catalog

import (
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog/internal/store"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog/usecase"

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
