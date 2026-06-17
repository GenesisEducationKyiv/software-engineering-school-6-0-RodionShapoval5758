package fanout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	batchSize    = 100
	pollInterval = time.Second
)

type recipientLister interface {
	ListConfirmed(ctx context.Context, repoID int64) ([]Recipient, error)
}

type Worker struct {
	pool       *pgxpool.Pool
	recipients recipientLister
}

func NewWorker(pool *pgxpool.Pool, recipients recipientLister) *Worker {
	return &Worker{pool: pool, recipients: recipients}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.processPending(ctx); err != nil {
				slog.Error("fanout worker error", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) processPending(ctx context.Context) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	releases, err := FetchForUpdate(ctx, tx, batchSize)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		return nil
	}

	var processed []int64

	for _, r := range releases {
		recs, err := w.recipients.ListConfirmed(ctx, r.RepoID)
		if err != nil {
			slog.Error("fanout: list confirmed recipients", "repo_id", r.RepoID, "error", err)
			continue
		}

		payloads, err := buildEvents(r, recs)
		if err != nil {
			slog.Error("fanout: build events", "repo_id", r.RepoID, "error", err)
			continue
		}

		failed := false
		for _, payload := range payloads {
			if err := outbox.Insert(ctx, tx, contract.SubjectRelease, payload); err != nil {
				slog.Error("fanout: insert outbox", "repo_id", r.RepoID, "error", err)
				failed = true
			}
		}

		if !failed {
			processed = append(processed, r.ID)
		}
	}

	if len(processed) == 0 {
		return nil
	}

	if err := MarkProcessed(ctx, tx, processed); err != nil {
		return err
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
