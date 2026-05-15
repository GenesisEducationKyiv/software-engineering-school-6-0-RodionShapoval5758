package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"GithubReleaseNotificationAPI/internal/domain"
	gh "GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/store"
)

type SubscriptionService interface {
	Subscribe(ctx context.Context, email string, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error)
}

type githubClient interface {
	CheckRepo(ctx context.Context, fullName string) error
}

type subscriptionRepository interface {
	Create(ctx context.Context, subscription domain.Subscription) error
	FindByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error)
	Confirm(ctx context.Context, token string) error
	DeleteByUnsubscribeToken(ctx context.Context, token string) error
	HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error)
	ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error)
}

type repositoryRepository interface {
	Create(ctx context.Context, repositoryName string) (*domain.Repository, error)
	FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error)
	DeleteByID(ctx context.Context, repositoryID int64) error
}

type smtpClient interface {
	SendConfirmationEmail(toEmail, repoName, confirmToken string) error
}

type subscriptionService struct {
	subscriptionRepository subscriptionRepository
	repositoryRepository   repositoryRepository
	githubClient           githubClient
	smtpClient             smtpClient
}

func NewSubscriptionService(
	subscriptionRepository subscriptionRepository,
	repositoryRepository repositoryRepository,
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

const maxTokenGenerationAttempts = 5

func (s *subscriptionService) Subscribe(ctx context.Context, email string, repo string) error {
	email, repo, err := normalizeSubscriptionInput(email, repo)
	if err != nil {
		return err
	}

	if err := s.verifyRepositoryExists(ctx, repo); err != nil {
		return err
	}

	repoDomain, err := s.ensureRepository(ctx, repo)
	if err != nil {
		return err
	}

	token, err := s.createPendingSubscription(ctx, email, repoDomain.ID)
	if err != nil {
		return err
	}

	if err := s.sendConfirmationEmail(email, repo, token); err != nil {
		return err
	}

	return nil
}

func (s *subscriptionService) Confirm(ctx context.Context, token string) error {
	return s.confirmSubscription(ctx, token)
}

func (s *subscriptionService) Unsubscribe(ctx context.Context, token string) error {
	subscriptionDomain, err := s.findSubscriptionByUnsubscribeToken(ctx, token)
	if err != nil {
		return err
	}

	if err := s.deleteSubscriptionByUnsubscribeToken(ctx, token); err != nil {
		return err
	}

	return s.cleanupRepositoryIfOrphaned(ctx, subscriptionDomain.RepositoryID)
}

func (s *subscriptionService) ListByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error) {
	email = strings.TrimSpace(email)
	if err := validateEmailFormat(email); err != nil {
		return nil, err
	}

	subscriptions, err := s.subscriptionRepository.ListSubscriptionDetailsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions by email: %w", err)
	}

	return subscriptions, nil
}

func normalizeSubscriptionInput(email string, repo string) (string, string, error) {
	email = strings.TrimSpace(email)
	if err := validateEmailFormat(email); err != nil {
		return "", "", err
	}

	repo = strings.TrimSpace(repo)
	if err := validateRepoFormat(repo); err != nil {
		return "", "", err
	}

	return email, repo, nil
}

func (s *subscriptionService) verifyRepositoryExists(ctx context.Context, repo string) error {
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

	return nil
}

func (s *subscriptionService) ensureRepository(ctx context.Context, repo string) (*domain.Repository, error) {
	repoDomain, err := s.repositoryRepository.FindByFullName(ctx, repo)
	if err == nil {
		return repoDomain, nil
	}

	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("find repository %s: %w", repo, err)
	}

	return s.createRepositoryWithConflictRecovery(ctx, repo)
}

func (s *subscriptionService) createRepositoryWithConflictRecovery(ctx context.Context, repo string) (*domain.Repository, error) {
	repoDomain, err := s.repositoryRepository.Create(ctx, repo)
	if err == nil {
		return repoDomain, nil
	}

	if !errors.Is(err, store.ErrAlreadyExists) {
		return nil, fmt.Errorf("create repository %s: %w", repo, err)
	}

	repoDomain, err = s.repositoryRepository.FindByFullName(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("find repository %s after create conflict: %w", repo, err)
	}

	return repoDomain, nil
}

func (s *subscriptionService) createPendingSubscription(ctx context.Context, email string, repositoryID int64) (string, error) {
	for range maxTokenGenerationAttempts {
		sub, err := domain.NewSubscription(email, repositoryID)
		if err != nil {
			return "", fmt.Errorf("prepare domain subscription: %w", err)
		}

		err = s.subscriptionRepository.Create(ctx, *sub)
		if errors.Is(err, store.ErrTokensAlreadyExists) {
			continue
		}

		if err != nil {
			switch {
			case errors.Is(err, store.ErrAlreadyExists):
				return "", ErrSubscriptionAlreadyExists
			case errors.Is(err, store.ErrTokensAlreadyExists):
				return "", fmt.Errorf("create subscription tokens conflict after retries: %w", err)
			default:
				return "", fmt.Errorf("failed to create subscription: %w", err)
			}
		}

		return sub.ConfirmToken, nil
	}

	return "", fmt.Errorf("create subscription tokens conflict after retries: %w", store.ErrTokensAlreadyExists)
}

func (s *subscriptionService) sendConfirmationEmail(email string, repo string, token string) error {
	if err := s.smtpClient.SendConfirmationEmail(email, repo, token); err != nil {
		return fmt.Errorf("send confirmation email for repo %s: %w", repo, err)
	}

	return nil
}

func (s *subscriptionService) confirmSubscription(ctx context.Context, token string) error {
	if err := s.subscriptionRepository.Confirm(ctx, token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("confirm token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("confirm subscription: %w", err)
	}

	return nil
}

func (s *subscriptionService) findSubscriptionByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error) {
	subscriptionDomain, err := s.subscriptionRepository.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return nil, fmt.Errorf("find subscription by unsubscribe token: %w", err)
	}

	return subscriptionDomain, nil
}

func (s *subscriptionService) deleteSubscriptionByUnsubscribeToken(ctx context.Context, token string) error {
	if err := s.subscriptionRepository.DeleteByUnsubscribeToken(ctx, token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("unsubscribe token not found: %w", ErrTokenNotFound)
		}

		return fmt.Errorf("delete subscription by unsubscribe token: %w", err)
	}

	return nil
}

func (s *subscriptionService) cleanupRepositoryIfOrphaned(ctx context.Context, repositoryID int64) error {
	hasAnySubscriptions, err := s.subscriptionRepository.HasAnyByRepositoryID(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("check remaining subscriptions for repository_id %d: %w", repositoryID, err)
	}

	if hasAnySubscriptions {
		return nil
	}

	if err := s.repositoryRepository.DeleteByID(ctx, repositoryID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("repository %d disappeared during unsubscribe cleanup: %w", repositoryID, err)
		}

		return fmt.Errorf("delete orphaned repository %d: %w", repositoryID, err)
	}

	return nil
}
