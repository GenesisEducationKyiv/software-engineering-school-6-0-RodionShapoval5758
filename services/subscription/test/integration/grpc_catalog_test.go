//go:build integration

package integration_test

import (
	"context"
	"sort"

	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"
)

func (s *IntegrationSuite) TestGRPCCatalog_ListTrackedRepos_ReturnsSeededRepos() {
	id1 := s.seedRepository("owner/repo-a")
	id2 := s.seedRepository("owner/repo-b")

	client := catalogv1.NewCatalogServiceClient(s.grpcConn)
	resp, err := client.ListTrackedRepos(context.Background(), &catalogv1.ListTrackedReposRequest{})

	s.Require().NoError(err)
	s.Require().Len(resp.Repos, 2)

	sort.Slice(resp.Repos, func(i, j int) bool {
		return resp.Repos[i].RepoId < resp.Repos[j].RepoId
	})
	s.Equal(id1, resp.Repos[0].RepoId)
	s.Equal("owner/repo-a", resp.Repos[0].FullName)
	s.Equal(id2, resp.Repos[1].RepoId)
	s.Equal("owner/repo-b", resp.Repos[1].FullName)
}

func (s *IntegrationSuite) TestGRPCCatalog_ListTrackedRepos_EmptyDB() {
	client := catalogv1.NewCatalogServiceClient(s.grpcConn)
	resp, err := client.ListTrackedRepos(context.Background(), &catalogv1.ListTrackedReposRequest{})

	s.Require().NoError(err)
	s.Empty(resp.Repos)
}
