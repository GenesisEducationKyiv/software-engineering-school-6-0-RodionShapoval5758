package fanout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
)

type recipientLister interface {
	ListConfirmed(ctx context.Context, repoID int64) ([]Recipient, error)
}

type outboxWriter interface {
	Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error
}

// DetectedRelease is the per-release data used by buildEvents.
type DetectedRelease struct {
	ReleaseTag  string
	ReleaseName string
	ReleaseURL  string
}

type Worker struct {
	js         jetstream.JetStream
	pool       *pgxpool.Pool
	recipients recipientLister
	outbox     outboxWriter
}

func NewWorker(js jetstream.JetStream, pool *pgxpool.Pool, recipients recipientLister, outbox outboxWriter) *Worker {
	return &Worker{js: js, pool: pool, recipients: recipients, outbox: outbox}
}

func (w *Worker) Run(ctx context.Context) error {
	cons, err := w.js.CreateOrUpdateConsumer(ctx, contract.StreamName, jetstream.ConsumerConfig{
		Durable:       "fanout-consumer",
		FilterSubject: contract.SubjectReleaseFound,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err := w.handleReleaseFound(ctx, msg); err != nil {
			slog.Error("fanout handle release found", "error", err)
			if err := msg.Nak(); err != nil {
				slog.Error("fanout nak failed", "error", err)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("fanout ack failed", "error", err)
		}
	})
	if err != nil {
		return err
	}
	defer cc.Stop()

	<-ctx.Done()
	return nil
}

func (w *Worker) handleReleaseFound(ctx context.Context, msg jetstream.Msg) error {
	var ev contract.ReleaseFound
	if err := json.Unmarshal(msg.Data(), &ev); err != nil {
		slog.Error("fanout: unmarshal ReleaseFound", "error", err)
		return nil
	}

	recs, err := w.recipients.ListConfirmed(ctx, ev.RepoID)
	if err != nil {
		return fmt.Errorf("fanout: list confirmed for repo_id=%d: %w", ev.RepoID, err)
	}

	if len(recs) == 0 {
		return nil
	}

	dr := DetectedRelease{
		ReleaseTag:  ev.ReleaseTag,
		ReleaseName: ev.ReleaseName,
		ReleaseURL:  ev.ReleaseURL,
	}

	payloads, err := buildEvents(dr, recs)
	if err != nil {
		return fmt.Errorf("fanout: build events: %w", err)
	}

	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("fanout: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, payload := range payloads {
		if err := w.outbox.Insert(ctx, tx, contract.SubjectRelease, payload); err != nil {
			return fmt.Errorf("fanout: insert outbox: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func buildEvents(r DetectedRelease, recs []Recipient) ([][]byte, error) {
	payloads := make([][]byte, 0, len(recs))

	for _, rec := range recs {
		ev := contract.ReleaseDetected{
			Email:            rec.Email,
			UnsubscribeToken: rec.UnsubscribeToken,
			ReleaseTag:       r.ReleaseTag,
			ReleaseName:      r.ReleaseName,
			ReleaseURL:       r.ReleaseURL,
		}

		data, err := json.Marshal(ev)
		if err != nil {
			return nil, fmt.Errorf("marshal release event: %w", err)
		}

		payloads = append(payloads, data)
	}

	return payloads, nil
}
