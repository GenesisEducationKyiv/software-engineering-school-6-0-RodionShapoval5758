package monitoring

import (
	"context"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/github"
)

type catalogClient interface {
	ListTracked(ctx context.Context) ([]catalog.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repositoryID int64, tag string) error
	UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error
}

type githubClient interface {
	GetLatestTag(ctx context.Context, fullName string) (*github.Release, error)
}

type outboxWriter interface {
	Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error
}

type scanObserver interface {
	ObserveScanDuration(seconds float64)
	IncScanResult(result string)
}
