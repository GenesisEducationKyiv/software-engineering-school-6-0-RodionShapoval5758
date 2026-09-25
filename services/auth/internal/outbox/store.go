package outbox

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/services/auth/internal/db"

	"github.com/jackc/pgx/v5"
)

type Row struct {
	ID      int64
	Subject string
	Payload []byte
}

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error {
	_, err := q.Exec(ctx, insertOutboxQuery, subject, payload)
	if err != nil {
		return fmt.Errorf("insert outbox row: %w", err)
	}

	return nil
}

func (s *Store) FetchForUpdate(ctx context.Context, tx pgx.Tx, limit int) ([]Row, error) {
	rows, err := tx.Query(ctx, fetchOutboxForUpdateQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox rows: %w", err)
	}
	defer rows.Close()

	var result []Row

	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Subject, &r.Payload); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}

		result = append(result, r)
	}

	return result, rows.Err()
}

func (s *Store) MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error {
	_, err := tx.Exec(ctx, markOutboxPublishedQuery, ids)
	if err != nil {
		return fmt.Errorf("mark outbox rows published: %w", err)
	}

	return nil
}
