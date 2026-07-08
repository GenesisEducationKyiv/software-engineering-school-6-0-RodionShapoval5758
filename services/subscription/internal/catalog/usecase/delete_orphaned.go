package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"
)

type deleteStore interface {
	DeleteByID(ctx context.Context, repositoryID int64) error
}

type outboxWriter interface {
	Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error
}

type DeleteIfOrphaned struct {
	store  deleteStore
	pool   db.DBTX
	outbox outboxWriter
}

func NewDeleteIfOrphaned(store deleteStore, pool db.DBTX, outbox outboxWriter) *DeleteIfOrphaned {
	return &DeleteIfOrphaned{store: store, pool: pool, outbox: outbox}
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

	payload, err := json.Marshal(contract.RepoUntracked{RepoID: repoID})
	if err == nil {
		if err := uc.outbox.Insert(ctx, uc.pool, contract.SubjectRepoUntracked, payload); err != nil {
			slog.Error("emit RepoUntracked", "repo_id", repoID, "error", err)
		}
	}

	return nil
}
