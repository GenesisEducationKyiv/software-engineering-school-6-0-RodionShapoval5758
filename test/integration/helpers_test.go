//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

type fakeGithubClient struct{}

func (f *fakeGithubClient) CheckRepo(_ context.Context, _ string) error { return nil }

func (s *IntegrationSuite) do(method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *IntegrationSuite) seedRepository(fullName string) int64 {
	var id int64
	err := testPool.QueryRow(
		context.Background(),
		"INSERT INTO repositories (name) VALUES ($1) RETURNING id",
		fullName,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *IntegrationSuite) seedSubscription(email, confirmToken, unsubscribeToken string, repoID int64, confirmed bool) {
	_, err := testPool.Exec(
		context.Background(),
		`INSERT INTO subscriptions (email, repository_id, confirmation_token, unsubscribe_token, confirmed)
		 VALUES ($1, $2, $3, $4, $5)`,
		email, repoID, confirmToken, unsubscribeToken, confirmed,
	)
	s.Require().NoError(err)
}

func mailpitCount() int {
	resp, err := http.Get("http://localhost:8025/api/v1/messages")
	if err != nil {
		return -1
	}
	defer resp.Body.Close()
	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Total
}

func clearMailpit() {
	req, _ := http.NewRequest(http.MethodDelete, "http://localhost:8025/api/v1/messages", nil)
	http.DefaultClient.Do(req)
}
