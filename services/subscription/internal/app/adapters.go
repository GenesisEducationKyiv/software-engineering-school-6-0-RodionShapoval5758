package app

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/services/subscription/internal/fanout"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
)

type dbPinger struct{ pool *pgxpool.Pool }

func (d *dbPinger) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

type natsPinger struct{ nc *natsgo.Conn }

func (n *natsPinger) Ping(_ context.Context) error {
	if n.nc.Status() != natsgo.CONNECTED {
		return errors.New("not connected")
	}
	return nil
}

type confirmedLister interface {
	ConfirmedByRepositoryID(ctx context.Context, repoID int64) ([]subscription.Subscription, error)
}

type recipientListerAdapter struct {
	lister confirmedLister
}

func (a *recipientListerAdapter) ListConfirmed(ctx context.Context, repoID int64) ([]fanout.Recipient, error) {
	subs, err := a.lister.ConfirmedByRepositoryID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	return subsToRecipients(subs), nil
}

func subsToRecipients(subs []subscription.Subscription) []fanout.Recipient {
	recs := make([]fanout.Recipient, len(subs))
	for i, s := range subs {
		recs[i] = fanout.Recipient{Email: s.Email, UnsubscribeToken: s.UnsubscribeToken}
	}
	return recs
}
