//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"strings"
)

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

func (s *IntegrationSuite) TestListSubscriptions_Validation() {
	cases := []struct {
		name  string
		query string
	}{
		{"missing email param", ""},
		{"invalid email format", "?email=not-an-email"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			w := s.do(http.MethodGet, "/api/subscriptions"+tc.query, nil)
			s.Equal(http.StatusBadRequest, w.Code)
		})
	}
}
