package notifier_test

import (
	"context"

	"GithubReleaseNotificationAPI/internal/domain"

	"github.com/stretchr/testify/mock"
)

type mockSmtpClient struct {
	mock.Mock
}

func (m *mockSmtpClient) SendReleaseNotification(toEmail, unsubscribeToken string, release *domain.Release) error {
	args := m.Called(toEmail, unsubscribeToken, release)
	return args.Error(0)
}

type mockSubscriptionRepository struct {
	mock.Mock
}

func (m *mockSubscriptionRepository) ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Subscription), args.Error(1)
}

type mockGithubClient struct {
	mock.Mock
}

func (m *mockGithubClient) GetLatestTag(ctx context.Context, fullName string) (*domain.Release, error) {
	args := m.Called(ctx, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Release), args.Error(1)
}

type mockRepositoryRepository struct {
	mock.Mock
}

func (m *mockRepositoryRepository) ListTracked(ctx context.Context) ([]domain.Repository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Repository), args.Error(1)
}

func (m *mockRepositoryRepository) UpdateLastSeenTag(ctx context.Context, repositoryID int64, tag string) error {
	args := m.Called(ctx, repositoryID, tag)
	return args.Error(0)
}

type mockReleaseNotifier struct {
	mock.Mock
}

func (m *mockReleaseNotifier) NotifyConfirmedSubscribers(ctx context.Context, repo domain.Repository, release *domain.Release) error {
	args := m.Called(ctx, repo, release)
	return args.Error(0)
}
