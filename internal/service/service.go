package service

import (
	"GithubReleaseNotificationAPI/internal/domain"
	gh "GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/store"
	"GithubReleaseNotificationAPI/internal/store/repository"
	"GithubReleaseNotificationAPI/internal/store/subscription"
	"context"
	"errors"
	"fmt"
	"strings"
)

type SubscriptionService interface {
	Subscribe(ctx context.Context, email string, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]subscription.Details, error)
}

type githubClient interface {
	CheckRepo(ctx context.Context, fullName string) error
}

type smtpClient interface {
	SendConfirmationEmail(toEmail, repoName, confirmToken string) error
}

type subscriptionService struct {
	subscriptionRepository subscription.Repository
	repositoryRepository   repository.Repository
	githubClient           githubClient
	smtpClient             smtpClient
}

func NewSubscriptionService(
	subscriptionRepository subscription.Repository,
	repositoryRepository repository.Repository,
	githubClient githubClient,
	smtpClient smtpClient,
) SubscriptionService {
	return &subscriptionService{
		subscriptionRepository: subscriptionRepository,
		repositoryRepository:   repositoryRepository,
		githubClient:           githubClient,
		smtpClient:             smtpClient,
	}
}

const TokenLength = 32
const maxTokenGenerationAttempts = 5

func (s *subscriptionService) Subscribe(ctx context.Context, email string, repo string) error {
	email = strings.TrimSpace(email)
	if err := validateEmailFormat(email); err != nil {
		return err
	}

	repo = strings.TrimSpace(repo)
	if err := validateRepoFormat(repo); err != nil {
		return err
	}

	if err := s.githubClient.CheckRepo(ctx, repo); err != nil {
		switch {
		case errors.Is(err, gh.ErrNotFound):
			return ErrRepoNotFound
		case errors.Is(err, gh.ErrRateLimited):
			return ErrTooMuchRequests
		case errors.Is(err, gh.ErrUnexpectedResponse):
			return fmt.Errorf("github repo check failed: %w", err)
		default:
			return fmt.Errorf("github repo check request failed: %w", err)
		}
	}

	// Manual race condition handling
	repoDomain, err := s.repositoryRepository.FindByFullName(ctx, repo)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			repoDomain, err = s.repositoryRepository.Create(ctx, repo)
			if err != nil {
				if errors.Is(err, store.ErrAlreadyExists) {
					repoDomain, err = s.repositoryRepository.FindByFullName(ctx, repo)
					if err != nil {
						return fmt.Errorf("find repository %s after create conflict: %w", repo, err)
					}
				} else {
					return fmt.Errorf("create repository %s: %w", repo, err)
				}
			}
		} else {
			return fmt.Errorf("find repository %s: %w", repo, err)
		}
	}

	token, err := s.createSubscriptionWithGeneratedTokens(ctx, email, repoDomain.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrAlreadyExists):
			return ErrSubscriptionAlreadyExists
		case errors.Is(err, store.ErrTokensAlreadyExists):
			return fmt.Errorf("create subscription tokens conflict after retries: %w", err)
		default:
			return fmt.Errorf("failed to create subscription: %w", err)
		}
	}

	if err := s.smtpClient.SendConfirmationEmail(email, repo, token); err != nil {
		return fmt.Errorf("send email to %s for repo %s and with token %s: %w",
			email, repo, token, err)
	}

	return nil
}

func (s *subscriptionService) Confirm(ctx context.Context, token string) error {
	if err := s.subscriptionRepository.Confirm(ctx, token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("confirm token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("confirm subscription with token %s: %w", token, err)
	}

	return nil
}

func (s *subscriptionService) Unsubscribe(ctx context.Context, token string) error {
	subscriptionDomain, err := s.subscriptionRepository.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("find subscription with unsubscribe token %s: %w", token, err)
	}

	if err := s.subscriptionRepository.DeleteByUnsubscribeToken(ctx, token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("delete subscription with token %s: %w", token, err)
	}

	hasAnySubscriptions, err := s.subscriptionRepository.HasAnyByRepositoryID(ctx, subscriptionDomain.RepositoryID)
	if err != nil {
		return fmt.Errorf("check remaining subscriptions for repository_id %d: %w", subscriptionDomain.RepositoryID, err)
	}

	if hasAnySubscriptions {
		return nil
	}

	err = s.repositoryRepository.DeleteByID(ctx, subscriptionDomain.RepositoryID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("repository %d disappeared during unsubscribe cleanup: %w", subscriptionDomain.RepositoryID, err)
		}

		return fmt.Errorf("delete orphaned repository %d: %w", subscriptionDomain.RepositoryID, err)
	}

	return nil
}

func (s *subscriptionService) ListByEmail(ctx context.Context, email string) ([]subscription.Details, error) {
	email = strings.TrimSpace(email)
	err := validateEmailFormat(email)
	if err != nil {
		return nil, err
	}

	subscriptions, err := s.subscriptionRepository.ListSubscriptionDetailsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions with email %s: %w", email, err)
	}

	return subscriptions, nil
}

func (s *subscriptionService) createSubscriptionWithGeneratedTokens(ctx context.Context, email string, repositoryID int64) (string, error) {
	for range maxTokenGenerationAttempts {
		confirmToken, unsubscribeToken, err := GenerateTokens()
		if err != nil {
			return "", err
		}

		subscriptionInput := domain.Subscription{
			Email:            email,
			RepositoryID:     repositoryID,
			ConfirmToken:     confirmToken,
			UnsubscribeToken: unsubscribeToken,
		}

		err = s.subscriptionRepository.Create(ctx, subscriptionInput)
		if errors.Is(err, store.ErrTokensAlreadyExists) {
			continue
		}

		return confirmToken, err
	}

	return "", store.ErrTokensAlreadyExists
}

func GenerateTokens() (string, string, error) {
	token1, err := GenerateToken(TokenLength)
	if err != nil {
		return "", "", fmt.Errorf("create token: %w", err)
	}

	token2, err := GenerateToken(TokenLength)
	if err != nil {
		return "", "", fmt.Errorf("create token: %w", err)
	}

	return token1, token2, nil
}
