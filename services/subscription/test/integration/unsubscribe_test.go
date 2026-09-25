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

func (s *IntegrationSuite) TestUnsubscribe_Validation() {
	cases := []struct {
		name  string
		token string
	}{
		{"token too short", "hahaha"},
		{"token one char", "h"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			w := s.do(http.MethodGet, "/api/unsubscribe/"+tc.token, nil)
			s.Equal(http.StatusBadRequest, w.Code)
		})
	}
}

func (s *IntegrationSuite) TestUnsubscribe_UnknownToken() {
	w := s.do(http.MethodGet, "/api/unsubscribe/unknown-token-xyz", nil)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *IntegrationSuite) TestUnsubscribe_LastSubscriberCleansUpRepository() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, true)

	w := s.do(http.MethodGet, "/api/unsubscribe/unsub-token-12345678", nil)
	s.Equal(http.StatusOK, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM repositories WHERE id = $1",
		repoID,
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *IntegrationSuite) TestUnsubscribe_NotLastSubscriberKeepsRepository() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user1@example.com", "confirm-token-1", "unsub-token-11111111", repoID, true)
	s.seedSubscription("user2@example.com", "confirm-token-2", "unsub-token-22222222", repoID, true)

	w := s.do(http.MethodGet, "/api/unsubscribe/unsub-token-11111111", nil)
	s.Equal(http.StatusOK, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM repositories WHERE id = $1",
		repoID,
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count)
}
