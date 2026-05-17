//go:build integration

package integration_test

import (
	"context"
	"net/http"
)

func (s *IntegrationSuite) TestUnsubscribe_HappyPath() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, true)

	w := s.do(http.MethodGet, "/api/unsubscribe/unsub-token-12345678", nil)

	s.Equal(http.StatusOK, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM subscriptions WHERE unsubscribe_token = $1",
		"unsub-token-12345678",
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(0, count)
}
