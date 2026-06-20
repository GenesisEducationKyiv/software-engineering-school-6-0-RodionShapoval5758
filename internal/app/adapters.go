package app

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/fanout"
	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
)

type catalogLister interface {
	ListTracked(ctx context.Context) ([]catalog.Repository, error)
}

type catalogUpdater interface {
	UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error
}

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

type catalogMonitoringAdapter struct {
	lister  catalogLister
	updater catalogUpdater
}

func (a *catalogMonitoringAdapter) ListTracked(ctx context.Context) ([]monitoring.TrackedRepo, error) {
	repos, err := a.lister.ListTracked(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]monitoring.TrackedRepo, len(repos))
	for i, r := range repos {
		result[i] = monitoring.TrackedRepo{
			ID:          r.ID,
			FullName:    r.FullName,
			LastSeenTag: r.LastSeenTag,
		}
	}

	return result, nil
}

func (a *catalogMonitoringAdapter) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error {
	return a.updater.UpdateLastSeenTagAtomic(ctx, repoID, tag, onTx)
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
