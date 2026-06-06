package monitoring

import (
	"context"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/notification"
)

type catalogClient interface {
	ListTracked(ctx context.Context) ([]catalog.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repositoryID int64, tag string) error
}

type githubClient interface {
	GetLatestTag(ctx context.Context, fullName string) (*github.Release, error)
}

type subscriberReader interface {
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error)
}

type mailer interface {
	SendReleaseEmails(recipients []notification.ReleaseRecipient, release notification.ReleaseInfo) error
}

type scanObserver interface {
	ObserveScanDuration(seconds float64)
	IncScanResult(result string)
}
