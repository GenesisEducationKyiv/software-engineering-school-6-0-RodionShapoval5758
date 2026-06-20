package store

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"

	"github.com/jackc/pgx/v5"
)

const insertMonitoringOutboxQuery = `INSERT INTO monitoring_outbox (subject, payload) VALUES ($1, $2)`

const fetchMonitoringOutboxQuery = `
	SELECT id, subject, payload
	FROM monitoring_outbox
	WHERE published_at IS NULL
	ORDER BY id
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

const markMonitoringOutboxPublishedQuery = `
	UPDATE monitoring_outbox SET published_at = now() WHERE id = ANY($1)
`

type OutboxRow struct {
	ID      int64
	Subject string
	Payload []byte
}

type OutboxStore struct{}

func NewOutboxStore() *OutboxStore {
	return &OutboxStore{}
}

func (s *OutboxStore) Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error {
	_, err := q.Exec(ctx, insertMonitoringOutboxQuery, subject, payload)
	if err != nil {
		return fmt.Errorf("insert monitoring outbox: %w", err)
	}
	return nil
}

func (s *OutboxStore) FetchForUpdate(ctx context.Context, tx pgx.Tx, limit int) ([]OutboxRow, error) {
	rows, err := tx.Query(ctx, fetchMonitoringOutboxQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch monitoring outbox: %w", err)
	}
	defer rows.Close()

	var result []OutboxRow
	for rows.Next() {
		var r OutboxRow
		if err := rows.Scan(&r.ID, &r.Subject, &r.Payload); err != nil {
			return nil, fmt.Errorf("scan monitoring outbox row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *OutboxStore) MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error {
	_, err := tx.Exec(ctx, markMonitoringOutboxPublishedQuery, ids)
	if err != nil {
		return fmt.Errorf("mark monitoring outbox published: %w", err)
	}
	return nil
}
