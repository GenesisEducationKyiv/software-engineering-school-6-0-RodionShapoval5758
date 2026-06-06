package service

import (
	"context"

	"GithubReleaseNotificationAPI/internal/domain"
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
	SendConfirmation(toEmail, repoName, confirmToken string) error
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
