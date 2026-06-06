package store

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/catalog/internal/domain"
	"GithubReleaseNotificationAPI/internal/shared"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepoRepository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *PostgresRepoRepository {
	return &PostgresRepoRepository{pool: pool}
}

func (r *PostgresRepoRepository) Create(ctx context.Context, repositoryName string) (*domain.Repository, error) {
	var repo domain.Repository

	err := r.pool.QueryRow(ctx, createRepoQuery, repositoryName).Scan(
		&repo.ID,
		&repo.FullName,
		&repo.LastSeenTag,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return nil, shared.ErrAlreadyExists
		}

		return nil, fmt.Errorf("insert repository row with name %s: %w", repositoryName, err)
	}

	return &repo, nil
}

func (r *PostgresRepoRepository) FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error) {
	var repo domain.Repository

	err := r.pool.QueryRow(ctx, findByNameQuery, fullName).Scan(
		&repo.ID,
		&repo.FullName,
		&repo.LastSeenTag,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}

		return nil, fmt.Errorf("scan repositories row by name %s: %w", fullName, err)
	}

	return &repo, nil
}

func (r *PostgresRepoRepository) UpdateLastSeenTag(ctx context.Context, repositoryID int64, lastTag string) error {
	tag, err := r.pool.Exec(ctx, updateLastSeenTagByIDQuery, repositoryID, lastTag)
	if err != nil {
		return fmt.Errorf("update last_seen_tag %s in repo with id %d: %w", lastTag, repositoryID, err)
	}

	if tag.RowsAffected() == 0 {
		return shared.ErrNotFound
	}

	return nil
}

func (r *PostgresRepoRepository) DeleteByID(ctx context.Context, repositoryID int64) error {
	tag, err := r.pool.Exec(ctx, deleteByIDQuery, repositoryID)
	if err != nil {
		return fmt.Errorf("delete repository with id %d: %w", repositoryID, err)
	}

	if tag.RowsAffected() == 0 {
		return shared.ErrNotFound
	}

	return nil
}

func (r *PostgresRepoRepository) ListTracked(ctx context.Context) ([]domain.Repository, error) {
	rows, err := r.pool.Query(ctx, listTrackedReposQuery)
	if err != nil {
		return nil, fmt.Errorf("query tracked repositories: %w", err)
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		if err := rows.Scan(
			&repo.ID,
			&repo.FullName,
			&repo.LastSeenTag,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan repository row: %w", err)
		}
		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repository rows: %w", err)
	}

	return repos, nil
}
