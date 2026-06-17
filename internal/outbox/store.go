package outbox

import (
	"context"
	"fmt"

	"GithubReleaseNotificationAPI/internal/db"

	"github.com/jackc/pgx/v5"
)

type Row struct {
	ID      int64
	Subject string
	Payload []byte
}

// Insert enqueues an outbox event inside an existing transaction.
// The caller owns the transaction lifecycle (begin / commit / rollback).
func Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error {
	_, err := q.Exec(ctx, `INSERT INTO outbox (subject, payload) VALUES ($1, $2)`, subject, payload)
	if err != nil {
		return fmt.Errorf("insert outbox row: %w", err)
	}

	return nil
}

// FetchForUpdate selects up to limit pending rows and locks them so that
// concurrent relay instances skip already-claimed rows (SKIP LOCKED).
func FetchForUpdate(ctx context.Context, tx pgx.Tx, limit int) ([]Row, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, subject, payload
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
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

// MarkPublished stamps the given row IDs as published inside the same transaction.
func MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE outbox SET published_at = now() WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return fmt.Errorf("mark outbox rows published: %w", err)
	}

	return nil
}
