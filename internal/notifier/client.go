package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"GithubReleaseNotificationAPI/contract"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, http: httpClient}
}

func (c *Client) SendConfirmation(toEmail, repoName, confirmToken string) error {
	payload, err := json.Marshal(contract.ConfirmationRequested{
		Email:        toEmail,
		RepoName:     repoName,
		ConfirmToken: confirmToken,
	})
	if err != nil {
		return fmt.Errorf("marshal confirmation: %w", err)
	}

	return c.post("/v1/emails/confirmation", payload)
}

func (c *Client) SendReleaseEmails(recipients []ReleaseRecipient, release ReleaseInfo) error {
	var errs []error

	for _, r := range recipients {
		payload, err := json.Marshal(contract.ReleasePublished{
			Email:            r.Email,
			UnsubscribeToken: r.UnsubscribeToken,
			ReleaseTag:       release.Tag,
			ReleaseName:      release.Name,
			ReleaseURL:       release.URL,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("marshal release for %s: %w", r.Email, err))
			continue
		}

		if err := c.post("/v1/emails/release", payload); err != nil {
			errs = append(errs, fmt.Errorf("send release to %s: %w", r.Email, err))
		}
	}

	return errors.Join(errs...)
}

func (c *Client) post(path string, body []byte) error {
	resp, err := c.http.Post(c.baseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("post %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("post %s: unexpected status %d", path, resp.StatusCode)
	}

	return nil
}
