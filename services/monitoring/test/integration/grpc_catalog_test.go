//go:build integration

package integration_test

import (
	"context"
	"testing"

	"GithubReleaseNotificationAPI/services/monitoring/internal/catalogclient"
	"GithubReleaseNotificationAPI/services/monitoring/internal/store"
	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"

	"github.com/stretchr/testify/suite"
)

type CatalogClientSuite struct {
	suite.Suite
}

func TestCatalogClientSuite(t *testing.T) {
	suite.Run(t, new(CatalogClientSuite))
}

func (s *CatalogClientSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(), "TRUNCATE scan_cursors")
	s.Require().NoError(err)
	stubServer.setRepos(nil)
}

func (s *CatalogClientSuite) TestListTracked_WithCursors() {
	stubServer.setRepos([]*catalogv1.TrackedRepo{
		{RepoId: 10, FullName: "org/alpha"},
		{RepoId: 20, FullName: "org/beta"},
	})

	_, err := testPool.Exec(context.Background(),
		"INSERT INTO scan_cursors (repo_id, full_name, last_seen_tag) VALUES (10, 'org/alpha', 'v1.0.0')",
	)
	s.Require().NoError(err)

	adapter := catalogclient.New(store.NewCursorStore(testPool), catalogv1.NewCatalogServiceClient(testConn))

	repos, err := adapter.ListTracked(context.Background())
	s.Require().NoError(err)
	s.Require().Len(repos, 2)

	byID := make(map[int64]string, len(repos))
	for _, r := range repos {
		byID[r.ID] = r.LastSeenTag
	}

	s.Equal("v1.0.0", byID[10])
	s.Equal("", byID[20])
}

func (s *CatalogClientSuite) TestListTracked_NoCursors() {
	stubServer.setRepos([]*catalogv1.TrackedRepo{
		{RepoId: 99, FullName: "org/gamma"},
	})

	adapter := catalogclient.New(store.NewCursorStore(testPool), catalogv1.NewCatalogServiceClient(testConn))

	repos, err := adapter.ListTracked(context.Background())
	s.Require().NoError(err)
	s.Require().Len(repos, 1)
	s.Equal("", repos[0].LastSeenTag)
}
