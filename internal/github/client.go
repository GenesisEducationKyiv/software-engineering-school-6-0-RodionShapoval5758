package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	GithubAPI        = "https://api.github.com"
	githubAPIVersion = "2026-03-10"
	userAgent        = "GithubReleaseNotificationAPI"
)

type Service struct {
	client      *http.Client
	githubToken *string
	baseURL     string
}

type Release struct {
	Tag         string
	Name        string
	URL         string
	PublishedAt time.Time
}

type latestReleaseResponse struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

func NewGithubClient(cl *http.Client, token *string) *Service {
	return &Service{
		client:      cl,
		githubToken: token,
		baseURL:     GithubAPI,
	}
}

func (s *Service) CheckRepo(ctx context.Context, fullName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.doGet(ctx, "/repos/"+strings.TrimSpace(fullName))
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close github repository response body", "repository", fullName, "error", err)
		}
	}()

	if err := determineRepsonse(resp); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetLatestTag(ctx context.Context, fullName string) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.doGet(ctx, "/repos/"+strings.TrimSpace(fullName)+"/releases/latest")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close github latest release response body", "repository", fullName, "error", err)
		}
	}()

	if err := determineRepsonse(resp); err != nil {
		return nil, err
	}

	var githubRelease latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&githubRelease); err != nil {
		return nil, fmt.Errorf("decode latest github release response: %w", err)
	}

	return &Release{
		Tag:         githubRelease.TagName,
		Name:        githubRelease.Name,
		URL:         githubRelease.HTMLURL,
		PublishedAt: githubRelease.PublishedAt,
	}, nil
}

func (s *Service) doGet(ctx context.Context, path string) (*http.Response, error) {
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+path,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build github request: %w", err)
	}

	httpRequest.Header.Set("Accept", "application/vnd.github+json")
	httpRequest.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	httpRequest.Header.Set("User-Agent", userAgent)
	if s.githubToken != nil && *s.githubToken != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+*s.githubToken)
	}

	resp, err := s.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("perform github request: %w", err)
	}

	return resp, nil
}

func determineRepsonse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusForbidden, http.StatusTooManyRequests:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "" {
			return ErrRateLimited
		}
		return fmt.Errorf("%w: status %d", ErrUnexpectedResponse, resp.StatusCode)
	case http.StatusMovedPermanently:
		return fmt.Errorf("%w: status %d", ErrUnexpectedResponse, resp.StatusCode)
	default:
		return fmt.Errorf("%w: status %d", ErrUnexpectedResponse, resp.StatusCode)
	}
}
