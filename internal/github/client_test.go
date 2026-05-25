package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckRepoOK(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/golang/go", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	defer closeServer()

	err := client.CheckRepo(context.Background(), "golang/go")
	require.NoError(t, err)
}

func TestCheckRepoNotFound(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer closeServer()

	err := client.CheckRepo(context.Background(), "golang/go")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCheckRepoRateLimited(t *testing.T) {
	cases := []struct {
		name   string
		status int
		header string
		value  string
	}{
		{"403 + X-Ratelimit-Remaining", http.StatusForbidden, "X-Ratelimit-Remaining", "0"},
		{"403 + Retry-After", http.StatusForbidden, "Retry-After", "60"},
		{"429 + X-Ratelimit-Remaining", http.StatusTooManyRequests, "X-Ratelimit-Remaining", "0"},
		{"429 + Retry-After", http.StatusTooManyRequests, "Retry-After", "60"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(tc.header, tc.value)
				w.WriteHeader(tc.status)
			})
			defer closeServer()

			err := client.CheckRepo(context.Background(), "golang/go")
			require.ErrorIs(t, err, ErrRateLimited)
		})
	}
}

func TestCheckRepoUnauthorized(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer closeServer()

	err := client.CheckRepo(context.Background(), "golang/go")
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestGetLatestTagOK(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/golang/go/releases/latest", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tag_name":"v1.2.3",
			"name":"Release 1.2.3",
			"html_url":"https://github.com/golang/go/releases/tag/v1.2.3",
			"published_at":"2026-04-11T12:00:00Z"
		}`))
	})
	defer closeServer()

	release, err := client.GetLatestTag(context.Background(), "golang/go")
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", release.Tag)
	require.Equal(t, "Release 1.2.3", release.Name)
	require.Equal(t, "https://github.com/golang/go/releases/tag/v1.2.3", release.URL)
}

func newTestGithubClient(t *testing.T, handler http.HandlerFunc) (*Service, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := NewGithubClient(server.Client(), nil)
	client.baseURL = server.URL

	return client, server.Close
}
