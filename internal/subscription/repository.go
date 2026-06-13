package subscription

import (
	"GithubReleaseNotificationAPI/internal/subscription/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository = store.PostgresSubscriptionRepository

func NewRepository(pool *pgxpool.Pool) *Repository {
	return store.New(pool)
}
