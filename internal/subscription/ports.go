package subscription

import (
	"context"

	"GithubReleaseNotificationAPI/internal/subscription/internal/domain"
)

type catalogClient interface {
	Ensure(ctx context.Context, fullName string) (int64, error)
	DeleteIfOrphaned(ctx context.Context, repoID int64, hasSubscribers func(ctx context.Context, repoID int64) (bool, error)) error
}

type githubClient interface {
	CheckRepo(ctx context.Context, fullName string) error
}

type subscriptionRepository interface {
	Create(ctx context.Context, subscription domain.Subscription) error
	FindByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error)
	Confirm(ctx context.Context, token string) error
	DeleteByUnsubscribeToken(ctx context.Context, token string) error
	HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error)
	ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error)
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error)
}

type notifier interface {
	SendConfirmation(toEmail, repoName, confirmToken string) error
}
