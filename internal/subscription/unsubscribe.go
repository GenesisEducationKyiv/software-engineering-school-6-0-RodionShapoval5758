package subscription

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/shared"
)

func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	sub, err := s.subscriptionRepository.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("find subscription by unsubscribe token: %w", err)
	}

	if err := s.subscriptionRepository.DeleteByUnsubscribeToken(ctx, token); err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("delete subscription by unsubscribe token: %w", err)
	}

	return s.catalogClient.DeleteIfOrphaned(ctx, sub.RepositoryID, s.subscriptionRepository.HasAnyByRepositoryID)
}
