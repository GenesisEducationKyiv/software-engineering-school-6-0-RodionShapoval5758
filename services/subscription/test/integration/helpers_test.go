//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http/httptest"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type fakeGithubClient struct {
	err error
}

func (f *fakeGithubClient) CheckRepo(_ context.Context, _ string) error { return f.err }

func (s *IntegrationSuite) authToken(email string) string {
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub":   "test-user",
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})

	signed, err := tok.SignedString(s.signKey)
	s.Require().NoError(err)

	return signed
}

func (s *IntegrationSuite) do(method, path string, body io.Reader) *httptest.ResponseRecorder {
	return s.doWithAuth(method, path, body, "Bearer "+s.authToken("user@example.com"))
}

func (s *IntegrationSuite) doWithAuth(method, path string, body io.Reader, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
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
