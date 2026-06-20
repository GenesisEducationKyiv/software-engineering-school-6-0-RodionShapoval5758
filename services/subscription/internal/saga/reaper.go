package saga

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	reaperInterval = 30 * time.Second
	reaperBatch    = 50
)

type Reaper struct {
	pool         *pgxpool.Pool
	store        *Store
	orchestrator *Orchestrator
}

func NewReaper(pool *pgxpool.Pool, store *Store, o *Orchestrator) *Reaper {
	return &Reaper{pool: pool, store: store, orchestrator: o}
}

func (r *Reaper) Run(ctx context.Context) {
	ticker := time.NewTicker(reaperInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.reap(ctx); err != nil {
				slog.Error("saga reaper error", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *Reaper) reap(ctx context.Context) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	expired, err := r.store.FindExpiredForUpdate(ctx, tx, reaperBatch)
	if err != nil {
		return err
	}

	if len(expired) == 0 {
		return nil
	}

	for i := range expired {
		if err := r.store.UpdateState(ctx, tx, expired[i].ID, StateCompensating, nil); err != nil {
			slog.Error("reaper: mark COMPENSATING failed", "saga_id", expired[i].SagaID, "error", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	for _, row := range expired {
		if err := r.orchestrator.CompensateExpired(ctx, row); err != nil {
			slog.Error("reaper: compensation failed", "saga_id", row.SagaID, "error", err)
		}
	}

	return nil
}
