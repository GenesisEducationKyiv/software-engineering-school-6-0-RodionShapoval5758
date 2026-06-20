package usecase

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/db"
)

type deleteStore interface {
	DeleteByID(ctx context.Context, repositoryID int64) error
}

type DeleteIfOrphaned struct {
	store deleteStore
}

func NewDeleteIfOrphaned(store deleteStore) *DeleteIfOrphaned {
	return &DeleteIfOrphaned{store: store}
}

func (uc *DeleteIfOrphaned) DeleteIfOrphaned(ctx context.Context, repoID int64, hasSubscribers func(context.Context, int64) (bool, error)) error {
	has, err := hasSubscribers(ctx, repoID)
	if err != nil {
		return fmt.Errorf("check remaining subscriptions for repository_id %d: %w", repoID, err)
	}

	if has {
		return nil
	}

	if err := uc.store.DeleteByID(ctx, repoID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("repository %d disappeared during cleanup: %w", repoID, err)
		}

		return fmt.Errorf("delete orphaned repository %d: %w", repoID, err)
	}

	return nil
}
