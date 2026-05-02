package repository

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
	Create(ctx context.Context, repositoryName string) (*domain.Repository, error)
	FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repositoryID int64, tag string) error
	DeleteByID(ctx context.Context, repositoryID int64) error
	ListTracked(ctx context.Context) ([]domain.Repository, error)
}

type PostgresRepositoryRepository struct {
	pool *pgxpool.Pool
}

func NewRepositoryRepository(pool *pgxpool.Pool) *PostgresRepositoryRepository {
	return &PostgresRepositoryRepository{
		pool: pool,
	}
}

func (r *PostgresRepositoryRepository) Create(ctx context.Context, repositoryName string) (*domain.Repository, error) {
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
			return nil, store.ErrAlreadyExists
		}

		return nil, fmt.Errorf("insert repository row with name %s: %w", repositoryName, err)
	}

	return &repo, nil
}

func (r *PostgresRepositoryRepository) FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error) {
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
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("scan repositories row by name %s: %w", fullName, err)
	}

	return &repo, nil
}

func (r *PostgresRepositoryRepository) UpdateLastSeenTag(ctx context.Context, repositoryID int64, lastTag string) error {
	tag, err := r.pool.Exec(ctx, updateLastSeenTagByIDQuery, repositoryID, lastTag)
	if err != nil {
		return fmt.Errorf("update last_seen_tag %s in repo with id %d: %w", lastTag, repositoryID, err)
	}

	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}

	return nil
}

func (r *PostgresRepositoryRepository) DeleteByID(ctx context.Context, repositoryID int64) error {
	tag, err := r.pool.Exec(ctx, deleteByIDQuery, repositoryID)
	if err != nil {
		return fmt.Errorf("delete repository with id %d: %w", repositoryID, err)
	}

	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}

	return nil
}

func (r *PostgresRepositoryRepository) ListTracked(ctx context.Context) ([]domain.Repository, error) {
	rows, err := r.pool.Query(ctx, listTrackedReposQuery)
	if err != nil {
		return nil, fmt.Errorf("query tracked repositories: %w", err)
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		err := rows.Scan(
			&repo.ID,
			&repo.FullName,
			&repo.LastSeenTag,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan repository row: %w", err)
		}
		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repository rows: %w", err)
	}

	return repos, nil
}
