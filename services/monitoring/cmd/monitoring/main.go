package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GithubReleaseNotificationAPI/contract"
	monconfig "GithubReleaseNotificationAPI/services/monitoring/internal/config"
	monconsumer "GithubReleaseNotificationAPI/services/monitoring/internal/consumer"
	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/github"
	"GithubReleaseNotificationAPI/services/monitoring/internal/monitoring"
	monrelay "GithubReleaseNotificationAPI/services/monitoring/internal/relay"
	"GithubReleaseNotificationAPI/services/monitoring/internal/store"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With(slog.Group("service", slog.String("name", "monitoring-service")))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := monconfig.Load()
	if err != nil {
		return err
	}

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.RunMigrationsFrom(cfg.MigrationDSN(), "file://migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	pool, err := db.NewPool(initCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	nc, err := natsgo.Connect(cfg.NATSUrl)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("create jetstream: %w", err)
	}

	cursorStore := store.NewCursorStore(pool)
	outboxStore := store.NewOutboxStore()

	githubClient := github.NewGithubClient(&http.Client{Timeout: 15 * time.Second}, &cfg.GithubToken)

	catalogAdapter := &cursorCatalogAdapter{cursors: cursorStore}
	enqueuer := &releaseFoundEnqueuer{outbox: outboxStore}
	worker := monitoring.NewWorker(githubClient, catalogAdapter, enqueuer, nil)

	trackingConsumer := monconsumer.New(js, cursorStore, pool)
	relay := monrelay.New(pool, js, outboxStore)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("monitoring service started", "scan_interval", cfg.ScanInterval)

	go relay.Run(ctx)
	go func() {
		if err := trackingConsumer.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("tracking consumer error", "error", err)
		}
	}()

	if err := worker.Start(ctx, cfg.ScanInterval); err != nil {
		return err
	}

	if err := nc.Drain(); err != nil {
		slog.Error("nats drain failed", "error", err)
	}
	return nil
}

type cursorCatalogAdapter struct {
	cursors *store.CursorStore
}

func (a *cursorCatalogAdapter) ListTracked(ctx context.Context) ([]monitoring.TrackedRepo, error) {
	return a.cursors.ListTracked(ctx)
}

func (a *cursorCatalogAdapter) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, tag string, onTx func(context.Context, db.DBTX) error) error {
	return a.cursors.UpdateLastSeenTagAtomic(ctx, repoID, tag, onTx)
}

type releaseFoundEnqueuer struct {
	outbox *store.OutboxStore
}

func (e *releaseFoundEnqueuer) Enqueue(ctx context.Context, q db.DBTX, dr monitoring.DetectedRelease) error {
	payload, err := json.Marshal(contract.ReleaseFound{
		RepoID:      dr.RepoID,
		RepoName:    dr.RepoName,
		ReleaseTag:  dr.ReleaseTag,
		ReleaseName: dr.ReleaseName,
		ReleaseURL:  dr.ReleaseURL,
	})
	if err != nil {
		return fmt.Errorf("marshal ReleaseFound: %w", err)
	}

	return e.outbox.Insert(ctx, q, contract.SubjectReleaseFound, payload)
}
