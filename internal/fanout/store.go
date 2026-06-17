package fanout

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/internal/db"

	"github.com/jackc/pgx/v5"
)

type DetectedRelease struct {
	ID          int64
	RepoID      int64
	RepoName    string
	ReleaseTag  string
	ReleaseName string
	ReleaseURL  string
}

type Recipient struct {
	Email            string
	UnsubscribeToken string
}

func Enqueue(ctx context.Context, q db.DBTX, r DetectedRelease) error {
	_, err := q.Exec(ctx, `
		INSERT INTO detected_releases (repo_id, repo_name, release_tag, release_name, release_url)
		VALUES ($1, $2, $3, $4, $5)
	`, r.RepoID, r.RepoName, r.ReleaseTag, r.ReleaseName, r.ReleaseURL)
	if err != nil {
		return fmt.Errorf("enqueue detected release: %w", err)
	}

	return nil
}

func FetchForUpdate(ctx context.Context, tx pgx.Tx, limit int) ([]DetectedRelease, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, repo_id, repo_name, release_tag, release_name, release_url
		FROM detected_releases
		WHERE processed_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending detected releases: %w", err)
	}
	defer rows.Close()

	var result []DetectedRelease

	for rows.Next() {
		var r DetectedRelease
		if err := rows.Scan(&r.ID, &r.RepoID, &r.RepoName, &r.ReleaseTag, &r.ReleaseName, &r.ReleaseURL); err != nil {
			return nil, fmt.Errorf("scan detected release row: %w", err)
		}

		result = append(result, r)
	}

	return result, rows.Err()
}

func MarkProcessed(ctx context.Context, tx pgx.Tx, ids []int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE detected_releases SET processed_at = now() WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return fmt.Errorf("mark detected releases processed: %w", err)
	}

	return nil
}
