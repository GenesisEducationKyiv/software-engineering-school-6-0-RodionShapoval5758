package handler

import (
	"context"
	"errors"
	"testing"

	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCatalogLister struct {
	mock.Mock
}

func (m *mockCatalogLister) ListTracked(ctx context.Context) ([]catalog.Repository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]catalog.Repository), args.Error(1)
}

func TestCatalogHandler_ListTrackedRepos_MultipleRepos(t *testing.T) {
	lister := &mockCatalogLister{}
	repos := []catalog.Repository{
		{ID: 1, FullName: "owner/repo1"},
		{ID: 2, FullName: "owner/repo2"},
	}
	lister.On("ListTracked", mock.Anything).Return(repos, nil)

	h := NewCatalog(lister)
	resp, err := h.ListTrackedRepos(context.Background(), &catalogv1.ListTrackedReposRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Repos, 2)
	assert.Equal(t, int64(1), resp.Repos[0].RepoId)
	assert.Equal(t, "owner/repo1", resp.Repos[0].FullName)
	assert.Equal(t, int64(2), resp.Repos[1].RepoId)
	assert.Equal(t, "owner/repo2", resp.Repos[1].FullName)

	lister.AssertExpectations(t)
}

func TestCatalogHandler_ListTrackedRepos_EmptyList(t *testing.T) {
	lister := &mockCatalogLister{}
	lister.On("ListTracked", mock.Anything).Return(nil, nil)

	h := NewCatalog(lister)
	resp, err := h.ListTrackedRepos(context.Background(), &catalogv1.ListTrackedReposRequest{})

	require.NoError(t, err)
	assert.NotNil(t, resp.Repos)
	assert.Len(t, resp.Repos, 0)

	lister.AssertExpectations(t)
}

func TestCatalogHandler_ListTrackedRepos_ListerError(t *testing.T) {
	lister := &mockCatalogLister{}
	lister.On("ListTracked", mock.Anything).Return(nil, errors.New("db exploded"))

	h := NewCatalog(lister)
	resp, err := h.ListTrackedRepos(context.Background(), &catalogv1.ListTrackedReposRequest{})

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	lister.AssertExpectations(t)
}
