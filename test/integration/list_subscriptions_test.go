//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
)

func (s *IntegrationSuite) TestListSubscriptions_HappyPath() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, true)

	w := s.do(http.MethodGet, "/api/subscriptions?email=user@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 1)
	s.Equal("user@example.com", result[0]["email"])
	s.Equal("owner/repo", result[0]["repo"])
	s.True(result[0]["confirmed"].(bool))
}
