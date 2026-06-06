package service

import (
	"context"
	"fmt"
	"strings"

	"GithubReleaseNotificationAPI/internal/domain"
)

func (s *subscriptionService) ListByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error) {
	email = strings.TrimSpace(email)
	if err := domain.ValidateEmail(email); err != nil {
		return nil, err
	}

	subscriptions, err := s.subscriptionRepository.ListSubscriptionDetailsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions by email: %w", err)
	}

	return subscriptions, nil
}
