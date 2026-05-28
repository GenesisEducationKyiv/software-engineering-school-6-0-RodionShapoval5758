//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"strings"

	"GithubReleaseNotificationAPI/internal/domain"
)

func (s *IntegrationSuite) TestSubscribe_HappyPath() {
	body := `{"email":"user@example.com","repo":"owner/repo"}`
	w := s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body))

	s.Equal(http.StatusOK, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM subscriptions WHERE email = $1",
		"user@example.com",
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count)

	s.Equal(1, mailpitCount())
}

func (s *IntegrationSuite) TestSubscribe_Validation() {
	cases := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"missing email", `{"repo":"owner/repo"}`},
		{"missing repo", `{"email":"user@example.com"}`},
		{"invalid email format", `{"email":"not-an-email","repo":"owner/repo"}`},
		{"repo no slash", `{"email":"user@example.com","repo":"noslash"}`},
		{"repo too many slashes", `{"email":"user@example.com","repo":"a/b/c"}`},
		{"repo empty owner", `{"email":"user@example.com","repo":"/repo"}`},
		{"repo empty name", `{"email":"user@example.com","repo":"owner/"}`},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}

			w := s.do(http.MethodPost, "/api/subscribe", body)

			s.Equal(http.StatusBadRequest, w.Code)

			var count int
			err := testPool.QueryRow(
				context.Background(),
				"SELECT COUNT(*) FROM subscriptions",
			).Scan(&count)
			s.Require().NoError(err)
			s.Equal(0, count)
		})
	}
}

func (s *IntegrationSuite) TestSubscribe_DuplicateSubscription() {
	body := `{"email":"user@example.com","repo":"owner/repo"}`

	s.Require().Equal(http.StatusOK, s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body)).Code)

	s.Equal(http.StatusConflict, s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body)).Code)

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
	s.githubFake.err = domain.ErrNotFound

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
