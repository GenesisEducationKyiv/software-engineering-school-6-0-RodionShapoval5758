package service

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/domain"
)

func (s *subscriptionService) Unsubscribe(ctx context.Context, token string) error {
	sub, err := s.findSubscriptionByUnsubscribeToken(ctx, token)
	if err != nil {
		return err
	}

	if err := s.deleteSubscriptionByUnsubscribeToken(ctx, token); err != nil {
		return err
	}

	return s.cleanupRepositoryIfOrphaned(ctx, sub.RepositoryID)
}

func (s *subscriptionService) findSubscriptionByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error) {
	sub, err := s.subscriptionRepository.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return nil, fmt.Errorf("find subscription by unsubscribe token: %w", err)
	}

	return sub, nil
}

func (s *subscriptionService) deleteSubscriptionByUnsubscribeToken(ctx context.Context, token string) error {
	if err := s.subscriptionRepository.DeleteByUnsubscribeToken(ctx, token); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("delete subscription by unsubscribe token: %w", err)
	}

	return nil
}

func (s *subscriptionService) cleanupRepositoryIfOrphaned(ctx context.Context, repositoryID int64) error {
	hasAny, err := s.subscriptionRepository.HasAnyByRepositoryID(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("check remaining subscriptions for repository_id %d: %w", repositoryID, err)
	}

	if hasAny {
		return nil
	}

	if err := s.repositoryRepository.DeleteByID(ctx, repositoryID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("repository %d disappeared during unsubscribe cleanup: %w", repositoryID, err)
		}

		return fmt.Errorf("delete orphaned repository %d: %w", repositoryID, err)
	}

	return nil
}
