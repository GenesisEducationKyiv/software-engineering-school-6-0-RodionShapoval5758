package store

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/monitoring"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const listCursorsQuery = `
	SELECT repo_id, full_name, last_seen_tag FROM scan_cursors
`

const upsertCursorQuery = `
	INSERT INTO scan_cursors (repo_id, full_name)
	VALUES ($1, $2)
	ON CONFLICT (repo_id) DO UPDATE SET full_name = EXCLUDED.full_name
`

const deleteCursorQuery = `
	DELETE FROM scan_cursors WHERE repo_id = $1
`

const updateLastSeenTagQuery = `
	UPDATE scan_cursors SET last_seen_tag = $2, updated_at = now() WHERE repo_id = $1
`

type CursorStore struct {
	pool *pgxpool.Pool
}

func NewCursorStore(pool *pgxpool.Pool) *CursorStore {
	return &CursorStore{pool: pool}
}

func (s *CursorStore) ListTracked(ctx context.Context) ([]monitoring.TrackedRepo, error) {
	rows, err := s.pool.Query(ctx, listCursorsQuery)
	if err != nil {
		return nil, fmt.Errorf("list scan cursors: %w", err)
	}
	defer rows.Close()

	var result []monitoring.TrackedRepo
	for rows.Next() {
		var r monitoring.TrackedRepo
		if err := rows.Scan(&r.ID, &r.FullName, &r.LastSeenTag); err != nil {
			return nil, fmt.Errorf("scan cursor row: %w", err)
		}
		result = append(result, r)
	}

	return result, rows.Err()
}

func (s *CursorStore) Upsert(ctx context.Context, q db.DBTX, repoID int64, fullName string) error {
	_, err := q.Exec(ctx, upsertCursorQuery, repoID, fullName)
	if err != nil {
		return fmt.Errorf("upsert scan cursor repo_id=%d: %w", repoID, err)
	}
	return nil
}

func (s *CursorStore) Delete(ctx context.Context, q db.DBTX, repoID int64) error {
	_, err := q.Exec(ctx, deleteCursorQuery, repoID)
	if err != nil {
		return fmt.Errorf("delete scan cursor repo_id=%d: %w", repoID, err)
	}
	return nil
}

func (s *CursorStore) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, updateLastSeenTagQuery, repoID, tag); err != nil {
		return fmt.Errorf("update last_seen_tag repo_id=%d: %w", repoID, err)
	}

	if err := onTx(ctx, tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
