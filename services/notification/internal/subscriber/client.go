package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Subscriber struct {
	Email            string `json:"email"`
	UnsubscribeToken string `json:"unsubscribe_token"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, internalToken string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   internalToken,
		http:    &http.Client{Timeout: 5 * http.DefaultClient.Timeout},
	}
}

func (c *Client) ListConfirmed(ctx context.Context, repoID int64) ([]Subscriber, error) {
	url := fmt.Sprintf("%s/internal/repositories/%d/confirmed-subscribers", c.baseURL, repoID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call confirmed-subscribers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("confirmed-subscribers returned %d", resp.StatusCode)
	}

	var subs []Subscriber
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("decode confirmed-subscribers response: %w", err)
	}

	return subs, nil
}
