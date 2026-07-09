package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/services/auth/internal/db"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) CreateUser(ctx context.Context, q db.DBTX, email, passwordHash string) (string, error) {
	var id string

	err := q.QueryRow(ctx, insertUserQuery, email, passwordHash).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return "", ErrEmailTaken
		}

		return "", fmt.Errorf("insert user: %w", err)
	}

	return id, nil
}

func (s *Store) GetByEmail(ctx context.Context, q db.DBTX, email string) (User, error) {
	var u User

	err := q.QueryRow(ctx, getUserByEmailQuery, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by email: %w", err)
	}

	return u, nil
}

func (s *Store) InsertVerification(ctx context.Context, q db.DBTX, token, userID string, expiresAt time.Time) error {
	if _, err := q.Exec(ctx, insertVerificationQuery, token, userID, expiresAt); err != nil {
		return fmt.Errorf("insert email verification: %w", err)
	}

	return nil
}

type Verification struct {
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func (s *Store) GetVerificationForUpdate(ctx context.Context, q db.DBTX, token string) (Verification, error) {
	var v Verification

	err := q.QueryRow(ctx, getVerificationForUpdateQuery, token).
		Scan(&v.UserID, &v.ExpiresAt, &v.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Verification{}, ErrTokenNotFound
	}
	if err != nil {
		return Verification{}, fmt.Errorf("get email verification: %w", err)
	}

	return v, nil
}

func (s *Store) MarkVerificationUsed(ctx context.Context, q db.DBTX, token string) error {
	if _, err := q.Exec(ctx, markVerificationUsedQuery, token); err != nil {
		return fmt.Errorf("mark verification used: %w", err)
	}

	return nil
}

func (s *Store) MarkUserVerified(ctx context.Context, q db.DBTX, userID string) error {
	if _, err := q.Exec(ctx, markUserVerifiedQuery, userID); err != nil {
		return fmt.Errorf("mark user verified: %w", err)
	}

	return nil
}

func (s *Store) InsertRefreshToken(ctx context.Context, q db.DBTX, userID, tokenHash string, expiresAt time.Time) error {
	if _, err := q.Exec(ctx, insertRefreshTokenQuery, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}

	return nil
}

type RefreshToken struct {
	UserID    string
	Email     string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (s *Store) GetRefreshTokenForUpdate(ctx context.Context, q db.DBTX, tokenHash string) (RefreshToken, error) {
	var rt RefreshToken

	err := q.QueryRow(ctx, getRefreshTokenForUpdateQuery, tokenHash).
		Scan(&rt.UserID, &rt.ExpiresAt, &rt.RevokedAt, &rt.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshToken{}, ErrRefreshTokenInvalid
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("get refresh token: %w", err)
	}

	return rt, nil
}

func (s *Store) RevokeRefreshToken(ctx context.Context, q db.DBTX, tokenHash string) error {
	if _, err := q.Exec(ctx, revokeRefreshTokenQuery, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}
