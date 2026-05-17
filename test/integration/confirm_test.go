//go:build integration

package integration_test

import (
	"context"
	"net/http"
)

func (s *IntegrationSuite) TestConfirm_HappyPath() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, false)

	w := s.do(http.MethodGet, "/api/confirm/confirm-token-abc", nil)

	s.Equal(http.StatusOK, w.Code)

	var confirmed bool
	err := testPool.QueryRow(
		context.Background(),
		"SELECT confirmed FROM subscriptions WHERE confirmation_token = $1",
		"confirm-token-abc",
	).Scan(&confirmed)
	s.Require().NoError(err)
	s.True(confirmed)
}
