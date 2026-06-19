package subscription_test

import (
	"context"

	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

type mockSubscriber struct {
	mock.Mock
}

func (m *mockSubscriber) Execute(ctx context.Context, email, repo string) error {
	args := m.Called(ctx, email, repo)
	return args.Error(0)
}

type mockConfirmer struct {
	mock.Mock
}

func (m *mockConfirmer) Execute(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

type mockUnsubscriber struct {
	mock.Mock
}

func (m *mockUnsubscriber) Execute(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

type mockLister struct {
	mock.Mock
}

func (m *mockLister) ByEmail(ctx context.Context, email string) ([]subscription.SubscriptionDetails, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.SubscriptionDetails), args.Error(1)
}

type stubPinger struct{}

func (*stubPinger) Ping(_ context.Context) error { return nil }
