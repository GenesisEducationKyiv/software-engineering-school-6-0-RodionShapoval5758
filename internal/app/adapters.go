package app

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/outbox"

	natsgo "github.com/nats-io/nats.go"
)

// natsPinger wraps a NATS connection for the health check Pinger interface.
type natsPinger struct{ nc *natsgo.Conn }

func (n *natsPinger) Ping(_ context.Context) error {
	if n.nc.Status() != natsgo.CONNECTED {
		return errors.New("not connected")
	}
	return nil
}

// outboxStoreAdapter bridges the outbox package free functions to the
// outboxWriter interface consumed by the subscription service and worker.
type outboxStoreAdapter struct{}

func (o *outboxStoreAdapter) Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error {
	return outbox.Insert(ctx, q, subject, payload)
}
