//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	gh "GithubReleaseNotificationAPI/internal/github"
)

// --- Subscribe ---

func (s *IntegrationSuite) TestSubscribe_DuplicateSubscription() {
	body := `{"email":"user@example.com","repo":"owner/repo"}`

	s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body))
	w := s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body))

	s.Equal(http.StatusConflict, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM subscriptions WHERE email = $1",
		"user@example.com",
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count)
}

func (s *IntegrationSuite) TestSubscribe_RepoNotFoundOnGitHub() {
	s.githubFake.err = gh.ErrNotFound

	body := `{"email":"user@example.com","repo":"owner/repo"}`
	w := s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body))

	s.Equal(http.StatusNotFound, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM subscriptions",
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(0, count)
}

// --- Confirm ---

func (s *IntegrationSuite) TestConfirm_UnknownToken() {
	w := s.do(http.MethodGet, "/api/confirm/unknown-token", nil)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *IntegrationSuite) TestConfirm_AlreadyConfirmed() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, true)

	w := s.do(http.MethodGet, "/api/confirm/confirm-token-abc", nil)
	s.Equal(http.StatusOK, w.Code)
}

// --- Unsubscribe ---

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

// --- List ---

func (s *IntegrationSuite) TestListSubscriptions_NoSubscriptions() {
	w := s.do(http.MethodGet, "/api/subscriptions?email=user@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 0)
}

func (s *IntegrationSuite) TestListSubscriptions_UnconfirmedNotReturned() {
	repoID := s.seedRepository("owner/repo")
	s.seedSubscription("user@example.com", "confirm-token-abc", "unsub-token-12345678", repoID, false)

	w := s.do(http.MethodGet, "/api/subscriptions?email=user@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 0)
}

func (s *IntegrationSuite) TestListSubscriptions_MultipleRepos() {
	repoID1 := s.seedRepository("owner/repo1")
	repoID2 := s.seedRepository("owner/repo2")
	s.seedSubscription("user@example.com", "confirm-token-1", "unsub-token-11111111", repoID1, true)
	s.seedSubscription("user@example.com", "confirm-token-2", "unsub-token-22222222", repoID2, true)

	w := s.do(http.MethodGet, "/api/subscriptions?email=user@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	s.Require().NoError(err)
	s.Len(result, 2)
}
