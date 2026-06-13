package subscription

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"GithubReleaseNotificationAPI/internal/subscription/internal/domain"

	githubclient "GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/shared"
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

	token, err := s.createPendingSubscription(ctx, email, repositoryID)
	if err != nil {
		return err
	}

	return s.sendConfirmationEmail(email, repo, token)
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

func (s *Service) createPendingSubscription(ctx context.Context, email string, repositoryID int64) (string, error) {
	for range maxTokenGenerationAttempts {
		sub, err := domain.NewSubscription(email, repositoryID)
		if err != nil {
			return "", fmt.Errorf("prepare domain subscription: %w", err)
		}

		err = s.subscriptionRepository.Create(ctx, *sub)
		if errors.Is(err, shared.ErrTokenConflict) {
			continue
		}

		if err != nil {
			if errors.Is(err, shared.ErrAlreadyExists) {
				return "", ErrSubscriptionAlreadyExists
			}

			return "", fmt.Errorf("failed to create subscription: %w", err)
		}

		return sub.ConfirmToken, nil
	}

	return "", fmt.Errorf("create subscription tokens conflict after retries: %w", shared.ErrTokenConflict)
}

func (s *Service) sendConfirmationEmail(email, repo, token string) error {
	if err := s.notifier.SendConfirmation(email, repo, token); err != nil {
		return fmt.Errorf("send confirmation email for repo %s: %w", repo, err)
	}

	return nil
}
