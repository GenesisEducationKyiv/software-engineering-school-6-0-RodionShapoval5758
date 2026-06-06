package subscription

import (
	"context"
	"fmt"
	"strings"

	"GithubReleaseNotificationAPI/internal/subscription/internal/domain"
)

func (s *Service) ListByEmail(ctx context.Context, email string) ([]SubscriptionDetails, error) {
	email = strings.TrimSpace(email)
	if err := domain.ValidateEmail(email); err != nil {
		return nil, err
	}

	details, err := s.subscriptionRepository.ListSubscriptionDetailsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions by email: %w", err)
	}

	return details, nil
}

func (s *Service) ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]Subscription, error) {
	return s.subscriptionRepository.ListConfirmedByRepositoryID(ctx, repositoryID)
}
