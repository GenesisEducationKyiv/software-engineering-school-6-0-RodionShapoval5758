//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
)

func (s *IntegrationSuite) TestListSubscriptions_HappyPath() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, true)

	w := s.do(http.MethodGet, "/api/subscriptions", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]any
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 1)
	s.Equal("user@example.com", result[0]["email"])
	s.Equal("owner/repo", result[0]["repo"])
	s.True(result[0]["confirmed"].(bool))
}

func (s *IntegrationSuite) TestListSubscriptions_NoSubscriptions() {
	w := s.do(http.MethodGet, "/api/subscriptions", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]any
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 0)
}

func (s *IntegrationSuite) TestListSubscriptions_UnconfirmedNotReturned() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, false)

	w := s.do(http.MethodGet, "/api/subscriptions", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]any
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 0)
}

func (s *IntegrationSuite) TestListSubscriptions_MultipleRepos() {
	repoID1 := s.seedRepository("owner/repo1")
	repoID2 := s.seedRepository("owner/repo2")
	s.seedSubscription("user@example.com", "confirm-token-1", "unsub-token-11111111", repoID1, true)
	s.seedSubscription("user@example.com", "confirm-token-2", "unsub-token-22222222", repoID2, true)

	w := s.do(http.MethodGet, "/api/subscriptions", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]any
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 2)
}
