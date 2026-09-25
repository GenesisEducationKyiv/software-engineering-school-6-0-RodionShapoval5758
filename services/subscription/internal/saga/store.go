package saga

import (
	"context"
	"errors"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"

	"github.com/jackc/pgx/v5"
)

type State string

const (
	StateStarted              State = "STARTED"
	StateAwaitingConfirmation State = "AWAITING_CONFIRMATION"
	StateCompleted            State = "COMPLETED"
	StateCompensating         State = "COMPENSATING"
	StateFailed               State = "FAILED"
)

type Row struct {
	ID             int64
	SagaID         string
	SubscriptionID int64
	RepositoryID   int64
	Email          string
	State          State
	DeadlineAt     time.Time
	LastError      *string
}

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

const insertSagaQuery = `
	INSERT INTO subscribe_sagas
		(saga_id, subscription_id, repository_id, email, state, deadline_at)
	VALUES ($1, $2, $3, $4, $5, $6)
`

func (s *Store) InsertInTx(ctx context.Context, q db.DBTX, r Row) error {
	_, err := q.Exec(ctx, insertSagaQuery,
		r.SagaID, r.SubscriptionID, r.RepositoryID, r.Email, r.State, r.DeadlineAt,
	)
	if err != nil {
		return fmt.Errorf("insert saga: %w", err)
	}

	return nil
}

const findBySagaIDForUpdateQuery = `
	SELECT id, saga_id, subscription_id, repository_id, email, state, deadline_at, last_error
	FROM subscribe_sagas
	WHERE saga_id = $1
	FOR UPDATE
`

func (s *Store) FindBySagaIDForUpdate(ctx context.Context, q db.DBTX, sagaID string) (*Row, error) {
	r, err := scanRow(q.QueryRow(ctx, findBySagaIDForUpdateQuery, sagaID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, fmt.Errorf("find saga %s: %w", sagaID, err)
	}

	return r, nil
}

const findExpiredForUpdateQuery = `
	SELECT id, saga_id, subscription_id, repository_id, email, state, deadline_at, last_error
	FROM subscribe_sagas
	WHERE state IN ('STARTED', 'AWAITING_CONFIRMATION')
	  AND deadline_at < now()
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

func (s *Store) FindExpiredForUpdate(ctx context.Context, q db.DBTX, limit int) ([]Row, error) {
	rows, err := q.Query(ctx, findExpiredForUpdateQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("find expired sagas: %w", err)
	}
	defer rows.Close()

	var result []Row
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan saga row: %w", err)
		}
		result = append(result, *r)
	}

	return result, rows.Err()
}

const updateSagaStateQuery = `
	UPDATE subscribe_sagas
	SET state = $1, last_error = $2, updated_at = now()
	WHERE id = $3
`

func (s *Store) UpdateState(ctx context.Context, q db.DBTX, id int64, state State, lastError *string) error {
	_, err := q.Exec(ctx, updateSagaStateQuery, state, lastError, id)
	if err != nil {
		return fmt.Errorf("update saga state: %w", err)
	}
	return nil
}

const deleteUnconfirmedSubscriptionQuery = `
	DELETE FROM subscriptions
	WHERE id = $1 AND confirmed = FALSE
`

func (s *Store) DeleteUnconfirmedSubscription(ctx context.Context, q db.DBTX, subscriptionID int64) (bool, error) {
	tag, err := q.Exec(ctx, deleteUnconfirmedSubscriptionQuery, subscriptionID)
	if err != nil {
		return false, fmt.Errorf("delete unconfirmed subscription %d: %w", subscriptionID, err)
	}
	return tag.RowsAffected() > 0, nil
}

const findSubscriptionIDByTokenQuery = `
	SELECT id FROM subscriptions WHERE confirmation_token = $1
`

func (s *Store) FindSubscriptionIDByToken(ctx context.Context, q db.DBTX, token string) (int64, error) {
	var id int64
	err := q.QueryRow(ctx, findSubscriptionIDByTokenQuery, token).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, db.ErrNotFound
		}
		return 0, fmt.Errorf("find subscription by token: %w", err)
	}

	return id, nil
}

const confirmSubscriptionByIDQuery = `
	UPDATE subscriptions
	SET confirmed = TRUE, confirmed_at = NOW()
	WHERE id = $1
`

func (s *Store) ConfirmSubscription(ctx context.Context, q db.DBTX, subscriptionID int64) error {
	tag, err := q.Exec(ctx, confirmSubscriptionByIDQuery, subscriptionID)
	if err != nil {
		return fmt.Errorf("confirm subscription %d: %w", subscriptionID, err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}

	return nil
}

const findSagaBySubscriptionIDForUpdateQuery = `
	SELECT id, saga_id, subscription_id, repository_id, email, state, deadline_at, last_error
	FROM subscribe_sagas
	WHERE subscription_id = $1
	FOR UPDATE
`

func (s *Store) FindBySubscriptionIDForUpdate(ctx context.Context, q db.DBTX, subscriptionID int64) (*Row, error) {
	r, err := scanRow(q.QueryRow(ctx, findSagaBySubscriptionIDForUpdateQuery, subscriptionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, fmt.Errorf("find saga by subscription_id %d: %w", subscriptionID, err)
	}

	return r, nil
}

func scanRow(row pgx.Row) (*Row, error) {
	var r Row
	err := row.Scan(
		&r.ID,
		&r.SagaID,
		&r.SubscriptionID,
		&r.RepositoryID,
		&r.Email,
		&r.State,
		&r.DeadlineAt,
		&r.LastError,
	)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
