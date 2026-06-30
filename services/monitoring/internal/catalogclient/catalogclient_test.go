package catalogclient

import (
	"context"
	"errors"
	"strings"
	"testing"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/monitoring"
	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockGRPCClient struct {
	mock.Mock
}

func (m *mockGRPCClient) ListTrackedRepos(ctx context.Context, in *catalogv1.ListTrackedReposRequest, opts ...grpc.CallOption) (*catalogv1.ListTrackedReposResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalogv1.ListTrackedReposResponse), args.Error(1)
}

type mockCursorStore struct {
	mock.Mock
}

func (m *mockCursorStore) GetLastSeenTag(ctx context.Context, repoID int64) (string, error) {
	args := m.Called(ctx, repoID)
	return args.String(0), args.Error(1)
}

func (m *mockCursorStore) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, fullName, tag string, onTx func(context.Context, db.DBTX) error) error {
	args := m.Called(ctx, repoID, fullName, tag, onTx)
	return args.Error(0)
}

func TestListTracked_MultipleRepos(t *testing.T) {
	grpcMock := &mockGRPCClient{}
	cursorMock := &mockCursorStore{}

	resp := &catalogv1.ListTrackedReposResponse{
		Repos: []*catalogv1.TrackedRepo{
			{RepoId: 1, FullName: "owner/a"},
			{RepoId: 2, FullName: "owner/b"},
		},
	}

	grpcMock.On("ListTrackedRepos", mock.Anything, mock.Anything, mock.Anything).Return(resp, nil).Once()
	cursorMock.On("GetLastSeenTag", mock.Anything, int64(1)).Return("v1.0.0", nil).Once()
	cursorMock.On("GetLastSeenTag", mock.Anything, int64(2)).Return("v2.0.0", nil).Once()

	adapter := New(cursorMock, grpcMock)
	repos, err := adapter.ListTracked(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []monitoring.TrackedRepo{
		{ID: 1, FullName: "owner/a", LastSeenTag: "v1.0.0"},
		{ID: 2, FullName: "owner/b", LastSeenTag: "v2.0.0"},
	}, repos)
	grpcMock.AssertExpectations(t)
	cursorMock.AssertExpectations(t)
}

func TestListTracked_EmptyResponse(t *testing.T) {
	grpcMock := &mockGRPCClient{}
	cursorMock := &mockCursorStore{}

	grpcMock.On("ListTrackedRepos", mock.Anything, mock.Anything, mock.Anything).Return(&catalogv1.ListTrackedReposResponse{Repos: nil}, nil).Once()

	adapter := New(cursorMock, grpcMock)
	repos, err := adapter.ListTracked(context.Background())

	require.NoError(t, err)
	assert.Empty(t, repos)
	grpcMock.AssertExpectations(t)
	cursorMock.AssertNotCalled(t, "GetLastSeenTag")
}

func TestListTracked_GRPCError(t *testing.T) {
	grpcMock := &mockGRPCClient{}
	cursorMock := &mockCursorStore{}

	grpcMock.On("ListTrackedRepos", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("unavailable")).Once()

	adapter := New(cursorMock, grpcMock)
	repos, err := adapter.ListTracked(context.Background())

	assert.Nil(t, repos)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "list tracked repos via grpc"))
}

func TestListTracked_CursorError(t *testing.T) {
	grpcMock := &mockGRPCClient{}
	cursorMock := &mockCursorStore{}

	resp := &catalogv1.ListTrackedReposResponse{
		Repos: []*catalogv1.TrackedRepo{
			{RepoId: 1, FullName: "owner/a"},
			{RepoId: 2, FullName: "owner/b"},
		},
	}

	grpcMock.On("ListTrackedRepos", mock.Anything, mock.Anything, mock.Anything).Return(resp, nil).Once()
	cursorMock.On("GetLastSeenTag", mock.Anything, int64(1)).Return("", errors.New("db down")).Once()

	adapter := New(cursorMock, grpcMock)
	repos, err := adapter.ListTracked(context.Background())

	assert.Nil(t, repos)
	assert.EqualError(t, err, "db down")
	grpcMock.AssertExpectations(t)
	cursorMock.AssertExpectations(t)
	cursorMock.AssertNotCalled(t, "GetLastSeenTag", mock.Anything, int64(2))
}
