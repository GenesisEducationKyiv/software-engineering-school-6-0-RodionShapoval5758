package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/store"

	"github.com/nats-io/nats.go/jetstream"
)

type cursorStore interface {
	Upsert(ctx context.Context, q db.DBTX, repoID int64, fullName string) error
	Delete(ctx context.Context, q db.DBTX, repoID int64) error
}

type Consumer struct {
	js      jetstream.JetStream
	cursors cursorStore
	pool    db.DBTX
}

func New(js jetstream.JetStream, cursors *store.CursorStore, pool db.DBTX) *Consumer {
	return &Consumer{js: js, cursors: cursors, pool: pool}
}

func (c *Consumer) Start(ctx context.Context) error {
	if _, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     contract.StreamTracking,
		Subjects: []string{contract.SubjectTrackingAll},
		Storage:  jetstream.FileStorage,
	}); err != nil {
		return err
	}

	cons, err := c.js.CreateOrUpdateConsumer(ctx, contract.StreamTracking, jetstream.ConsumerConfig{
		Durable:       "monitoring-tracking-consumer",
		FilterSubject: contract.SubjectTrackingAll,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err := c.handle(ctx, msg); err != nil {
			slog.Error("tracking consumer handle error", "subject", msg.Subject(), "error", err)
			if err := msg.Nak(); err != nil {
				slog.Error("tracking consumer nak failed", "error", err)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("tracking consumer ack failed", "error", err)
		}
	})
	if err != nil {
		return err
	}
	defer cc.Stop()

	<-ctx.Done()
	return nil
}

func (c *Consumer) handle(ctx context.Context, msg jetstream.Msg) error {
	switch msg.Subject() {
	case contract.SubjectRepoTracked:
		var ev contract.RepoTracked
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			slog.Error("unmarshal RepoTracked", "error", err)
			return nil
		}
		return c.cursors.Upsert(ctx, c.pool, ev.RepoID, ev.FullName)

	case contract.SubjectRepoUntracked:
		var ev contract.RepoUntracked
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			slog.Error("unmarshal RepoUntracked", "error", err)
			return nil
		}
		return c.cursors.Delete(ctx, c.pool, ev.RepoID)

	default:
		slog.Warn("tracking consumer: unknown subject", "subject", msg.Subject())
		return nil
	}
}
