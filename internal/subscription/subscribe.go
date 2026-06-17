package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/internal/subscription/internal/domain"

	githubclient "GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/shared"

	"github.com/jackc/pgx/v5"
)

const maxTokenGenerationAttempts = 5

func (s *Service) Subscribe(ctx context.Context, email string, repo string) error {
	email, repo, err := normalizeSubscriptionInput(email, repo)
	if err != nil {
		return err
	}

	if err := s.verifyRepositoryExists(ctx, repo); err != nil {
		return err
	}

	repositoryID, err := s.catalogClient.Ensure(ctx, repo)
	if err != nil {
		return fmt.Errorf("ensure repository %s: %w", repo, err)
	}

	return s.createPendingSubscription(ctx, email, repo, repositoryID)
}

func normalizeSubscriptionInput(email, repo string) (string, string, error) {
	email = strings.TrimSpace(email)
	if err := domain.ValidateEmail(email); err != nil {
		return "", "", err
	}

	repo = strings.TrimSpace(repo)
	if err := domain.ValidateRepo(repo); err != nil {
		return "", "", ErrInvalidRepoFormat
	}

	return email, repo, nil
}

func (s *Service) verifyRepositoryExists(ctx context.Context, repo string) error {
	if err := s.githubClient.CheckRepo(ctx, repo); err != nil {
		switch {
		case errors.Is(err, shared.ErrNotFound):
			return ErrRepoNotFound
		case errors.Is(err, githubclient.ErrRateLimited):
			return ErrTooMuchRequests
		case errors.Is(err, githubclient.ErrUnauthorized):
			return ErrGitHubUnauthorized
		default:
			return fmt.Errorf("github repo check failed: %w", err)
		}
	}

	return nil
}

func (s *Service) createPendingSubscription(ctx context.Context, email, repoName string, repositoryID int64) error {
	for range maxTokenGenerationAttempts {
		sub, err := domain.NewSubscription(email, repositoryID)
		if err != nil {
			return fmt.Errorf("prepare domain subscription: %w", err)
		}

		payload, err := json.Marshal(contract.ConfirmationRequested{
			Email:        email,
			RepoName:     repoName,
			ConfirmToken: sub.ConfirmToken,
		})
		if err != nil {
			return fmt.Errorf("marshal confirmation event: %w", err)
		}

		err = s.createAndEnqueueConfirmation(ctx, *sub, payload)
		if errors.Is(err, shared.ErrTokenConflict) {
			continue
		}

		if err != nil {
			if errors.Is(err, shared.ErrAlreadyExists) {
				return ErrSubscriptionAlreadyExists
			}

			return fmt.Errorf("failed to create subscription: %w", err)
		}

		return nil
	}

	return fmt.Errorf("create subscription tokens conflict after retries: %w", shared.ErrTokenConflict)
}

// createAndEnqueueConfirmation inserts the subscription row and the outbox
// confirmation event in a single Postgres transaction.
func (s *Service) createAndEnqueueConfirmation(ctx context.Context, sub domain.Subscription, payload []byte) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.subscriptionRepository.CreateInTx(ctx, tx, sub); err != nil {
		return err
	}

	if err := s.outbox.Insert(ctx, tx, contract.SubjectConfirmation, payload); err != nil {
		return fmt.Errorf("enqueue confirmation event: %w", err)
	}

	return tx.Commit(ctx)
}
