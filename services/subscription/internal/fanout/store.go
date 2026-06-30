package fanout

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"
)

type Recipient struct {
	Email            string
	UnsubscribeToken string
}

type RepoStore struct{}

func NewRepoStore() *RepoStore { return &RepoStore{} }

func (s *RepoStore) UpdateLastSeenTag(ctx context.Context, q db.DBTX, repoID int64, tag string) error {
	if _, err := q.Exec(ctx, updateLastSeenTagQuery, repoID, tag); err != nil {
		return fmt.Errorf("update last_seen_tag repo_id=%d: %w", repoID, err)
	}
	return nil
}
