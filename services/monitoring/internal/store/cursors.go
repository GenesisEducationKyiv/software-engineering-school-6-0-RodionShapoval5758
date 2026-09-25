package store

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const getLastSeenTagQuery = `
	SELECT last_seen_tag FROM scan_cursors WHERE repo_id = $1
`

const upsertLastSeenTagQuery = `
	INSERT INTO scan_cursors (repo_id, full_name, last_seen_tag)
	VALUES ($1, $2, $3)
	ON CONFLICT (repo_id) DO UPDATE SET last_seen_tag = EXCLUDED.last_seen_tag, updated_at = now()
`

type CursorStore struct {
	pool *pgxpool.Pool
}

func NewCursorStore(pool *pgxpool.Pool) *CursorStore {
	return &CursorStore{pool: pool}
}

func (s *CursorStore) GetLastSeenTag(ctx context.Context, repoID int64) (string, error) {
	var tag string
	err := s.pool.QueryRow(ctx, getLastSeenTagQuery, repoID).Scan(&tag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get last_seen_tag repo_id=%d: %w", repoID, err)
	}

	return tag, nil
}

func (s *CursorStore) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, fullName, tag string, onTx func(context.Context, db.DBTX) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, upsertLastSeenTagQuery, repoID, fullName, tag); err != nil {
		return fmt.Errorf("upsert last_seen_tag repo_id=%d: %w", repoID, err)
	}

	if err := onTx(ctx, tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
