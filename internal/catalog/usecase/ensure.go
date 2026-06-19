package usecase

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/catalog/internal/domain"
	"GithubReleaseNotificationAPI/internal/db"
)

type ensureStore interface {
	Create(ctx context.Context, repositoryName string) (*domain.Repository, error)
	FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error)
}

type Ensure struct {
	store ensureStore
}

func NewEnsure(store ensureStore) *Ensure {
	return &Ensure{store: store}
}

func (uc *Ensure) Ensure(ctx context.Context, fullName string) (int64, error) {
	repo, err := uc.store.FindByFullName(ctx, fullName)
	if err == nil {
		return repo.ID, nil
	}

	if !errors.Is(err, db.ErrNotFound) {
		return 0, fmt.Errorf("find repository %s: %w", fullName, err)
	}

	repo, err = uc.store.Create(ctx, fullName)
	if err == nil {
		return repo.ID, nil
	}

	if !errors.Is(err, db.ErrAlreadyExists) {
		return 0, fmt.Errorf("create repository %s: %w", fullName, err)
	}

	repo, err = uc.store.FindByFullName(ctx, fullName)
	if err != nil {
		return 0, fmt.Errorf("find repository %s after create conflict: %w", fullName, err)
	}

	return repo.ID, nil
}
