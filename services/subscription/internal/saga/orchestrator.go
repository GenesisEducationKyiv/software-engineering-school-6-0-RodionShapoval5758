package saga

import (
	"context"
	"fmt"
	"log/slog"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"

	"github.com/jackc/pgx/v5"
)

type catalogCleaner interface {
	DeleteIfOrphaned(ctx context.Context, repoID int64, hasSubscribers func(context.Context, int64) (bool, error)) error
}

type subscriberChecker interface {
	HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error)
}

type Orchestrator struct {
	store      *Store
	pool       db.TxBeginner
	catalog    catalogCleaner
	subChecker subscriberChecker
}

func NewOrchestrator(store *Store, pool db.TxBeginner, catalog catalogCleaner, subChecker subscriberChecker) *Orchestrator {
	return &Orchestrator{store: store, pool: pool, catalog: catalog, subChecker: subChecker}
}

func (o *Orchestrator) HandleEmailSent(ctx context.Context, sagaID string) error {
	tx, err := o.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := o.store.FindBySagaIDForUpdate(ctx, tx, sagaID)
	if err != nil {
		return fmt.Errorf("lock saga %s: %w", sagaID, err)
	}

	if row.State != StateStarted {
		slog.Info("saga EmailSent: no-op (state mismatch)", "saga_id", sagaID, "state", row.State)
		return nil
	}

	if err := o.store.UpdateState(ctx, tx, row.ID, StateAwaitingConfirmation, nil); err != nil {
		return fmt.Errorf("advance saga %s to AWAITING_CONFIRMATION: %w", sagaID, err)
	}

	return tx.Commit(ctx)
}

func (o *Orchestrator) HandleEmailFailed(ctx context.Context, sagaID, reason string) error {
	tx, err := o.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := o.store.FindBySagaIDForUpdate(ctx, tx, sagaID)
	if err != nil {
		return fmt.Errorf("lock saga %s: %w", sagaID, err)
	}

	if row.State != StateStarted && row.State != StateAwaitingConfirmation {
		slog.Info("saga EmailFailed: no-op (already terminal)", "saga_id", sagaID, "state", row.State)
		return nil
	}

	if err := o.store.UpdateState(ctx, tx, row.ID, StateCompensating, &reason); err != nil {
		return fmt.Errorf("mark saga %s COMPENSATING: %w", sagaID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit COMPENSATING transition: %w", err)
	}

	return o.compensate(ctx, *row, reason)
}

func (o *Orchestrator) CompensateExpired(ctx context.Context, row Row) error {
	return o.compensate(ctx, row, "confirmation deadline exceeded")
}

func (o *Orchestrator) compensate(ctx context.Context, row Row, reason string) error {
	tx, err := o.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin compensation tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	deleted, err := o.store.DeleteUnconfirmedSubscription(ctx, tx, row.SubscriptionID)
	if err != nil {
		return fmt.Errorf("delete pending subscription %d: %w", row.SubscriptionID, err)
	}

	if !deleted {
		// User confirmed just before compensation ran; saga should be COMPLETED.
		slog.Info("saga compensation: subscription already confirmed, marking COMPLETED",
			"saga_id", row.SagaID,
			"subscription_id", row.SubscriptionID,
		)
		if err := o.store.UpdateState(ctx, tx, row.ID, StateCompleted, nil); err != nil {
			return fmt.Errorf("mark saga %s COMPLETED: %w", row.SagaID, err)
		}
		return tx.Commit(ctx)
	}

	if err := o.store.UpdateState(ctx, tx, row.ID, StateFailed, &reason); err != nil {
		return fmt.Errorf("mark saga %s FAILED: %w", row.SagaID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit compensation tx: %w", err)
	}

	if err := o.catalog.DeleteIfOrphaned(ctx, row.RepositoryID, o.subChecker.HasAnyByRepositoryID); err != nil {
		slog.Error("saga compensation: catalog cleanup failed",
			"saga_id", row.SagaID,
			"repo_id", row.RepositoryID,
			"error", err,
		)
	}

	slog.Info("saga compensated", "saga_id", row.SagaID, "reason", reason)
	return nil
}

func (o *Orchestrator) HandleConfirmed(ctx context.Context, subscriptionID int64) error {
	tx, err := o.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := o.store.FindBySubscriptionIDForUpdate(ctx, tx, subscriptionID)
	if err != nil {
		if err == db.ErrNotFound {
			return nil
		}
		return fmt.Errorf("lock saga for subscription %d: %w", subscriptionID, err)
	}

	if row.State == StateCompleted || row.State == StateFailed {
		return nil
	}

	if err := o.store.UpdateState(ctx, tx, row.ID, StateCompleted, nil); err != nil {
		return fmt.Errorf("mark saga %s COMPLETED: %w", row.SagaID, err)
	}

	return tx.Commit(ctx)
}
