package relay

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"GithubReleaseNotificationAPI/services/monitoring/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	batchSize    = 100
	pollInterval = time.Second
)

type Relay struct {
	pool  *pgxpool.Pool
	js    jetstream.JetStream
	store *store.OutboxStore
}

func New(pool *pgxpool.Pool, js jetstream.JetStream, s *store.OutboxStore) *Relay {
	return &Relay{pool: pool, js: js, store: s}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.publishPending(ctx); err != nil {
				slog.Error("monitoring outbox relay error", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *Relay) publishPending(ctx context.Context) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := r.store.FetchForUpdate(ctx, tx, batchSize)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	var published []int64

	for _, row := range rows {
		msgID := "mon-" + strconv.FormatInt(row.ID, 10)

		if _, err := r.js.Publish(ctx, row.Subject, row.Payload, jetstream.WithMsgID(msgID)); err != nil {
			slog.Error("monitoring relay publish failed", "outbox_id", row.ID, "subject", row.Subject, "error", err)
			continue
		}

		published = append(published, row.ID)
	}

	if len(published) == 0 {
		return nil
	}

	if err := r.store.MarkPublished(ctx, tx, published); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
