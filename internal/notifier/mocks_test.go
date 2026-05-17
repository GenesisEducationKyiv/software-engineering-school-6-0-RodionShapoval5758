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
