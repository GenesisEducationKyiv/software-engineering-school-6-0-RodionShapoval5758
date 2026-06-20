package monitoring

import (
	"context"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/github"
)

type catalogClient interface {
	ListTracked(ctx context.Context) ([]TrackedRepo, error)
	UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error
}

type githubClient interface {
	GetLatestTag(ctx context.Context, fullName string) (*github.Release, error)
}

type releaseEnqueuer interface {
	Enqueue(ctx context.Context, q db.DBTX, r DetectedRelease) error
}

type scanObserver interface {
	ObserveScanDuration(seconds float64)
	IncScanResult(result string)
}
