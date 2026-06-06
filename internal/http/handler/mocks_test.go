package handler_test

import (
	"context"

	"GithubReleaseNotificationAPI/internal/domain"

	"github.com/stretchr/testify/mock"
)

type mockSubscriptionService struct {
	mock.Mock
}

func (m *mockSubscriptionService) Subscribe(ctx context.Context, email, repo string) error {
	args := m.Called(ctx, email, repo)

	return args.Error(0)
}

func (m *mockSubscriptionService) Confirm(ctx context.Context, token string) error {
	args := m.Called(ctx, token)

	return args.Error(0)
}

func (m *mockSubscriptionService) Unsubscribe(ctx context.Context, token string) error {
	args := m.Called(ctx, token)

	return args.Error(0)
}

func (m *mockSubscriptionService) ListByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.SubscriptionDetails), args.Error(1)
}
