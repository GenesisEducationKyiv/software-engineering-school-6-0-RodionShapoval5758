package usecase

import (
	"context"

	"GithubReleaseNotificationAPI/services/subscription/internal/catalog/internal/domain"
)

type listStore interface {
	ListTracked(ctx context.Context) ([]domain.Repository, error)
}

type ListTracked struct {
	store listStore
}

func NewListTracked(store listStore) *ListTracked {
	return &ListTracked{store: store}
}

func (uc *ListTracked) ListTracked(ctx context.Context) ([]domain.Repository, error) {
	return uc.store.ListTracked(ctx)
}
