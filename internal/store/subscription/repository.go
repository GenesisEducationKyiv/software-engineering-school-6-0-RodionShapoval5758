package subscription

import (
	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/store"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, subscription domain.Subscription) error
	FindByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error)
	Confirm(ctx context.Context, token string) error
	DeleteByUnsubscribeToken(ctx context.Context, token string) error
	HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error)
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error)
	ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]Details, error)
}

type PostgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{
		pool: pool,
	}
}

func (r *PostgresSubscriptionRepository) Create(ctx context.Context, subscription domain.Subscription) error {
	tag, err := r.pool.Exec(
		ctx,
		createSubscriptionQuery,
		subscription.Email,
		subscription.RepositoryID,
		subscription.ConfirmToken,
		subscription.UnsubscribeToken,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			switch pgErr.ConstraintName {
			case "subscriptions_email_repository_id_key":
				return store.ErrAlreadyExists
			case "subscriptions_confirmation_token_key":
				return fmt.Errorf("confirmation token: %w", store.ErrTokensAlreadyExists)
			case "subscriptions_unsubscribe_token_key":
				return fmt.Errorf("unsubscribe token : %w", store.ErrTokensAlreadyExists)
			default:
				return fmt.Errorf("unexpected unique violation on subscriptions: %w", err)
			}
		}

		return fmt.Errorf(
			"insert subscription for email %s and repository_id %d: %w",
			subscription.Email,
			subscription.RepositoryID,
			err,
		)
	}

	if rowsAffected := tag.RowsAffected(); rowsAffected != 1 {
		return fmt.Errorf("insert subscription row: expected 1 affected row, got %d", rowsAffected)
	}

	return nil
}

func (r *PostgresSubscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error) {
	subscription, err := scanSubscription(
		r.pool.QueryRow(
			ctx,
			findSubscriptionByUnsubscribeTokenQuery,
			token,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("find subscription by unsubscribe token: %w", err)
	}

	return subscription, nil
}

func (r *PostgresSubscriptionRepository) Confirm(ctx context.Context, token string) error {
	tag, err := r.pool.Exec(
		ctx,
		confirmSubscriptionByTokenQuery,
		token,
	)
	if err != nil {
		return fmt.Errorf("confirm subscription by token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}

	return nil
}

func (r *PostgresSubscriptionRepository) DeleteByUnsubscribeToken(ctx context.Context, token string) error {
	tag, err := r.pool.Exec(
		ctx,
		deleteSubscriptionByUnsubscribeTokenQuery,
		token,
	)
	if err != nil {
		return fmt.Errorf("delete subscription by unsubscribe token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}

	return nil
}

func (r *PostgresSubscriptionRepository) HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error) {
	var hasAny bool

	err := r.pool.QueryRow(ctx, hasAnySubscriptionsByRepositoryIDQuery, repositoryID).Scan(&hasAny)
	if err != nil {
		return false, fmt.Errorf("check subscriptions for repository_id %d: %w", repositoryID, err)
	}

	return hasAny, nil
}

func (r *PostgresSubscriptionRepository) ListConfirmedByRepositoryID(
	ctx context.Context,
	repositoryID int64,
) ([]domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, listConfirmedSubscriptionsByRepositoryIDQuery, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("query confirmed subscriptions by repository_id %d: %w", repositoryID, err)
	}
	defer rows.Close()

	var subs []domain.Subscription
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription row: %w", err)
		}

		subs = append(subs, *subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscription rows: %w", err)
	}

	return subs, nil
}

func (r *PostgresSubscriptionRepository) ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]Details, error) {
	rows, err := r.pool.Query(ctx, listSubscriptionDetailsByEmailQuery, email)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions details by email %s: %w", email, err)
	}
	defer rows.Close()

	var details []Details
	for rows.Next() {
		var detail Details
		err := rows.Scan(
			&detail.Email,
			&detail.Repo,
			&detail.Confirmed,
			&detail.LastSeenTag,
		)
		if err != nil {
			return nil, fmt.Errorf("scan subscription row: %w", err)
		}
		details = append(details, detail)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscription rows: %w", err)
	}

	return details, nil
}

func scanSubscription(row pgx.Row) (*domain.Subscription, error) {
	var subscription domain.Subscription

	err := row.Scan(
		&subscription.ID,
		&subscription.Email,
		&subscription.RepositoryID,
		&subscription.Confirmed,
		&subscription.ConfirmToken,
		&subscription.UnsubscribeToken,
		&subscription.CreatedAt,
		&subscription.ConfirmedAt,
	)
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}
