package watcher_test

import (
	"context"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/notification"

	"github.com/stretchr/testify/mock"
)

type mockSmtpClient struct {
	mock.Mock
}

func (m *mockSmtpClient) SendReleaseEmails(recipients []notification.ReleaseRecipient, release notification.ReleaseInfo) error {
	args := m.Called(recipients, release)

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
