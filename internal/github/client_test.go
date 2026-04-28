package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckRepoOK(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/golang/go" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer closeServer()

	err := client.CheckRepo(context.Background(), "golang/go")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestCheckRepoNotFound(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer closeServer()

	err := client.CheckRepo(context.Background(), "golang/go")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCheckRepoRateLimited(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	})
	defer closeServer()

	err := client.CheckRepo(context.Background(), "golang/go")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestGetLatestTagOK(t *testing.T) {
	client, closeServer := newTestGithubClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/golang/go/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

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
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if release.Tag != "v1.2.3" {
		t.Fatalf("unexpected tag: %q", release.Tag)
	}
	if release.Name != "Release 1.2.3" {
		t.Fatalf("unexpected name: %q", release.Name)
	}
	if release.URL != "https://github.com/golang/go/releases/tag/v1.2.3" {
		t.Fatalf("unexpected url: %q", release.URL)
	}
}

func newTestGithubClient(t *testing.T, handler http.HandlerFunc) (*Service, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := NewGithubClient(server.Client(), nil)
	client.baseURL = server.URL

	return client, server.Close
}
