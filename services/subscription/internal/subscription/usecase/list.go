package usecase

import (
	"context"
	"fmt"
	"strings"

	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription/internal/domain"
)

type listRepository interface {
	ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error)
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error)
}

type List struct {
	repo listRepository
}

func NewList(repo listRepository) *List {
	return &List{repo: repo}
}

func (uc *List) ByEmail(ctx context.Context, email string) ([]subscription.SubscriptionDetails, error) {
	email = strings.TrimSpace(email)
	if err := domain.ValidateEmail(email); err != nil {
		return nil, err
	}

	details, err := uc.repo.ListSubscriptionDetailsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions by email: %w", err)
	}

	return details, nil
}

func (uc *List) ConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]subscription.Subscription, error) {
	return uc.repo.ListConfirmedByRepositoryID(ctx, repositoryID)
}
