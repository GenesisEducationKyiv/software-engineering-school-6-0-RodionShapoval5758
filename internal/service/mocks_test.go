package service_test

import (
	"context"

	"GithubReleaseNotificationAPI/internal/domain"

	"github.com/stretchr/testify/mock"
)

type mockSubscriptionRepository struct {
	mock.Mock
}

func (m *mockSubscriptionRepository) Create(ctx context.Context, subscription domain.Subscription) error {
	args := m.Called(ctx, subscription)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*domain.Subscription, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Subscription), args.Error(1)
}

func (m *mockSubscriptionRepository) Confirm(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) DeleteByUnsubscribeToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) HasAnyByRepositoryID(ctx context.Context, repositoryID int64) (bool, error) {
	args := m.Called(ctx, repositoryID)
	return args.Bool(0), args.Error(1)
}

func (m *mockSubscriptionRepository) ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.SubscriptionDetails), args.Error(1)
}

type mockRepositoryRepository struct {
	mock.Mock
}

func (m *mockRepositoryRepository) Create(ctx context.Context, repositoryName string) (*domain.Repository, error) {
	args := m.Called(ctx, repositoryName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Repository), args.Error(1)
}

func (m *mockRepositoryRepository) FindByFullName(ctx context.Context, fullName string) (*domain.Repository, error) {
	args := m.Called(ctx, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Repository), args.Error(1)
}

func (m *mockRepositoryRepository) DeleteByID(ctx context.Context, repositoryID int64) error {
	args := m.Called(ctx, repositoryID)
	return args.Error(0)
}

type mockGithubClient struct {
	mock.Mock
}

func (m *mockGithubClient) CheckRepo(ctx context.Context, fullName string) error {
	args := m.Called(ctx, fullName)
	return args.Error(0)
}

type mockSmtpClient struct {
	mock.Mock
}

func (m *mockSmtpClient) SendConfirmationEmail(toEmail, repoName, confirmToken string) error {
	args := m.Called(toEmail, repoName, confirmToken)
	return args.Error(0)
}
