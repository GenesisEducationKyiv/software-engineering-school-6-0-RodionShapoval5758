package subscription_test

import (
	"context"

	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

type mockSubscriptionRepository struct {
	mock.Mock
}

func (m *mockSubscriptionRepository) Create(ctx context.Context, s subscription.Subscription) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) CreateInTx(ctx context.Context, q db.DBTX, sub subscription.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*subscription.Subscription, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
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

func (m *mockSubscriptionRepository) ListSubscriptionDetailsByEmail(ctx context.Context, email string) ([]subscription.SubscriptionDetails, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.SubscriptionDetails), args.Error(1)
}

func (m *mockSubscriptionRepository) ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]subscription.Subscription, error) {
	args := m.Called(ctx, repositoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.Subscription), args.Error(1)
}

type mockCatalogClient struct {
	mock.Mock
}

func (m *mockCatalogClient) Ensure(ctx context.Context, fullName string) (int64, error) {
	args := m.Called(ctx, fullName)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockCatalogClient) DeleteIfOrphaned(ctx context.Context, repoID int64, hasSubscribers func(context.Context, int64) (bool, error)) error {
	args := m.Called(ctx, repoID, hasSubscribers)
	return args.Error(0)
}

type mockGithubClient struct {
	mock.Mock
}

func (m *mockGithubClient) CheckRepo(ctx context.Context, fullName string) error {
	args := m.Called(ctx, fullName)
	return args.Error(0)
}

type mockOutboxWriter struct {
	mock.Mock
}

func (m *mockOutboxWriter) Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error {
	args := m.Called(ctx, subject, payload)
	return args.Error(0)
}

type fakeTx struct{}

func (*fakeTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (*fakeTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { return nil, nil }
func (*fakeTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row        { return nil }
func (*fakeTx) Commit(_ context.Context) error                                { return nil }
func (*fakeTx) Rollback(_ context.Context) error                              { return nil }

type fakeTxBeginner struct{}

func (*fakeTxBeginner) BeginTx(_ context.Context, _ pgx.TxOptions) (db.Tx, error) {
	return &fakeTx{}, nil
}

type mockServiceForHandler struct {
	mock.Mock
}

func (m *mockServiceForHandler) Subscribe(ctx context.Context, email, repo string) error {
	args := m.Called(ctx, email, repo)
	return args.Error(0)
}

func (m *mockServiceForHandler) Confirm(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockServiceForHandler) Unsubscribe(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockServiceForHandler) ListByEmail(ctx context.Context, email string) ([]subscription.SubscriptionDetails, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.SubscriptionDetails), args.Error(1)
}

type stubPinger struct{}

func (*stubPinger) Ping(_ context.Context) error { return nil }
