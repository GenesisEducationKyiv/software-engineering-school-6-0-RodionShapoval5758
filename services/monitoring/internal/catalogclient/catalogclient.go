package catalogclient

import (
	"context"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/monitoring"
	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"

	"google.golang.org/grpc"
)

const grpcListTrackedTimeout = 5 * time.Second

type cursorStore interface {
	GetLastSeenTag(ctx context.Context, repoID int64) (string, error)
	UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, fullName, tag string, onTx func(context.Context, db.DBTX) error) error
}

type Adapter struct {
	cursors cursorStore
	grpc    catalogv1.CatalogServiceClient
}

func New(cursors cursorStore, grpc catalogv1.CatalogServiceClient) *Adapter {
	return &Adapter{cursors: cursors, grpc: grpc}
}

func (a *Adapter) ListTracked(ctx context.Context) ([]monitoring.TrackedRepo, error) {
	ctx, cancel := context.WithTimeout(ctx, grpcListTrackedTimeout)
	defer cancel()

	resp, err := a.grpc.ListTrackedRepos(ctx, &catalogv1.ListTrackedReposRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return nil, fmt.Errorf("list tracked repos via grpc: %w", err)
	}

	repos := make([]monitoring.TrackedRepo, 0, len(resp.Repos))
	for _, r := range resp.Repos {
		tag, err := a.cursors.GetLastSeenTag(ctx, r.RepoId)
		if err != nil {
			return nil, err
		}
		repos = append(repos, monitoring.TrackedRepo{
			ID:          r.RepoId,
			FullName:    r.FullName,
			LastSeenTag: tag,
		})
	}

	return repos, nil
}

func (a *Adapter) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, fullName, tag string, onTx func(context.Context, db.DBTX) error) error {
	return a.cursors.UpdateLastSeenTagAtomic(ctx, repoID, fullName, tag, onTx)
}
