package usecase

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription/internal/domain"
)

type unsubscribeRepository interface {
	FindByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error)
	DeleteByUnsubscribeToken(ctx context.Context, token string) error
	HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error)
}

type unsubscribeCatalog interface {
	DeleteIfOrphaned(ctx context.Context, repoID int64, hasSubscribers func(context.Context, int64) (bool, error)) error
}

type Unsubscribe struct {
	repo    unsubscribeRepository
	catalog unsubscribeCatalog
}

func NewUnsubscribe(repo unsubscribeRepository, catalog unsubscribeCatalog) *Unsubscribe {
	return &Unsubscribe{repo: repo, catalog: catalog}
}

func (uc *Unsubscribe) Execute(ctx context.Context, token string) error {
	sub, err := uc.repo.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", subscription.ErrTokenNotFound)
		}

		return fmt.Errorf("find subscription by unsubscribe token: %w", err)
	}

	if err := uc.repo.DeleteByUnsubscribeToken(ctx, token); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", subscription.ErrTokenNotFound)
		}

		return fmt.Errorf("delete subscription by unsubscribe token: %w", err)
	}

	return uc.catalog.DeleteIfOrphaned(ctx, sub.RepositoryID, uc.repo.HasAnyByRepositoryID)
}
