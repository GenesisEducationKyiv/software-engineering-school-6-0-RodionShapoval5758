package catalog

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/catalog/internal/domain"
	"GithubReleaseNotificationAPI/internal/catalog/internal/store"
	"GithubReleaseNotificationAPI/internal/shared"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository = domain.Repository

type repoStore interface {
	Create(ctx context.Context, repositoryName string) (*domain.Repository, error)
	FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error)
	DeleteByID(ctx context.Context, repositoryID int64) error
	ListTracked(ctx context.Context) ([]domain.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repositoryID int64, lastTag string) error
}

type Service struct {
	store repoStore
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{store: store.New(pool)}
}

func ValidateRepo(repo string) error {
	return domain.ValidateRepo(repo)
}

func (s *Service) Ensure(ctx context.Context, fullName string) (int64, error) {
	repo, err := s.store.FindByFullName(ctx, fullName)
	if err == nil {
		return repo.ID, nil
	}

	if !errors.Is(err, shared.ErrNotFound) {
		return 0, fmt.Errorf("find repository %s: %w", fullName, err)
	}

	repo, err = s.store.Create(ctx, fullName)
	if err == nil {
		return repo.ID, nil
	}

	if !errors.Is(err, shared.ErrAlreadyExists) {
		return 0, fmt.Errorf("create repository %s: %w", fullName, err)
	}

	repo, err = s.store.FindByFullName(ctx, fullName)
	if err != nil {
		return 0, fmt.Errorf("find repository %s after create conflict: %w", fullName, err)
	}

	return repo.ID, nil
}

func (s *Service) DeleteIfOrphaned(ctx context.Context, repoID int64, hasSubscribers func(ctx context.Context, repoID int64) (bool, error)) error {
	has, err := hasSubscribers(ctx, repoID)
	if err != nil {
		return fmt.Errorf("check remaining subscriptions for repository_id %d: %w", repoID, err)
	}

	if has {
		return nil
	}

	if err := s.store.DeleteByID(ctx, repoID); err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return fmt.Errorf("repository %d disappeared during cleanup: %w", repoID, err)
		}

		return fmt.Errorf("delete orphaned repository %d: %w", repoID, err)
	}

	return nil
}

func (s *Service) ListTracked(ctx context.Context) ([]domain.Repository, error) {
	return s.store.ListTracked(ctx)
}

func (s *Service) UpdateLastSeenTag(ctx context.Context, repoID int64, tag string) error {
	return s.store.UpdateLastSeenTag(ctx, repoID, tag)
}
