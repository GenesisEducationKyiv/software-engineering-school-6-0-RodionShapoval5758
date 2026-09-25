package authclient

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	authv1 "GithubReleaseNotificationAPI/services/auth/api/gen/authv1/auth/v1"

	"google.golang.org/grpc"
)

const (
	fetchTimeout    = 10 * time.Second
	refreshInterval = 15 * time.Minute
)

var ErrKeyUnavailable = errors.New("auth signing key not loaded")

// Client fetches the auth service's JWT public signing key over gRPC and
// caches it so token verification never blocks on a network call.
type Client struct {
	grpc authv1.AuthServiceClient

	mu  sync.RWMutex
	key crypto.PublicKey
}

func New(grpcClient authv1.AuthServiceClient) *Client {
	return &Client{grpc: grpcClient}
}

func (c *Client) Key() (crypto.PublicKey, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.key == nil {
		return nil, ErrKeyUnavailable
	}

	return c.key, nil
}

func (c *Client) Load(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	resp, err := c.grpc.GetSigningKey(ctx, &authv1.GetSigningKeyRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return fmt.Errorf("get signing key via grpc: %w", err)
	}

	key, err := parsePublicKeyPEM(resp.PublicKeyPem)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.key = key
	c.mu.Unlock()

	return nil
}

// Run refreshes the cached key periodically so an auth-side key rotation is
// picked up without restarting subscription.
func (c *Client) Run(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.Load(ctx); err != nil {
				slog.Error("refresh auth signing key failed", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func parsePublicKeyPEM(raw []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("parse auth signing key: invalid PEM")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse auth signing key: %w", err)
	}

	return key, nil
}
