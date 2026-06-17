package app

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/fanout"
	"GithubReleaseNotificationAPI/internal/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
)

type dbPinger struct{ pool *pgxpool.Pool }

func (d *dbPinger) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

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

type fanoutEnqueuerAdapter struct{}

func (f *fanoutEnqueuerAdapter) Enqueue(ctx context.Context, q db.DBTX, r fanout.DetectedRelease) error {
	return fanout.Enqueue(ctx, q, r)
}
