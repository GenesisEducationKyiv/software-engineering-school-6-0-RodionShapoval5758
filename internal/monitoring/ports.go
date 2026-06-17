package monitoring

import (
	"context"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/fanout"
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

type releaseEnqueuer interface {
	Enqueue(ctx context.Context, q db.DBTX, r fanout.DetectedRelease) error
}

type scanObserver interface {
	ObserveScanDuration(seconds float64)
	IncScanResult(result string)
}
