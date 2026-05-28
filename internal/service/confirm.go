package service

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/domain"
)

func (s *subscriptionService) Confirm(ctx context.Context, token string) error {
	if err := s.subscriptionRepository.Confirm(ctx, token); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("confirm token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("confirm subscription: %w", err)
	}

	return nil
}
