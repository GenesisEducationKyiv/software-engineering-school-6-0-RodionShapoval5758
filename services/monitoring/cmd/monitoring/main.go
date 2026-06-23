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
	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	"GithubReleaseNotificationAPI/services/monitoring/internal/github"
	"GithubReleaseNotificationAPI/services/monitoring/internal/monitoring"
	monrelay "GithubReleaseNotificationAPI/services/monitoring/internal/relay"
	"GithubReleaseNotificationAPI/services/monitoring/internal/store"
	subscriptionv1pb "GithubReleaseNotificationAPI/services/subscription/api/gen/subscriptionv1/subscription/v1"
	"GithubReleaseNotificationAPI/services/subscription/api/gen/subscriptionv1/subscription/v1/v1connect"

	"connectrpc.com/connect"
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

	grpcClient := v1connect.NewSubscriptionServiceClient(
		http.DefaultClient,
		cfg.SubscriptionGRPCAddr,
		connect.WithGRPC(),
	)

	githubClient := github.NewGithubClient(&http.Client{Timeout: 15 * time.Second}, &cfg.GithubToken)

	catalogAdapter := &cursorCatalogAdapter{cursors: cursorStore, grpc: grpcClient}
	enqueuer := &releaseFoundEnqueuer{outbox: outboxStore}
	worker := monitoring.NewWorker(githubClient, catalogAdapter, enqueuer, nil)

	relay := monrelay.New(pool, js, outboxStore)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("monitoring service started", "scan_interval", cfg.ScanInterval)

	go relay.Run(ctx)

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
	grpc    v1connect.SubscriptionServiceClient
}

func (a *cursorCatalogAdapter) ListTracked(ctx context.Context) ([]monitoring.TrackedRepo, error) {
	resp, err := a.grpc.ListTrackedRepos(ctx, connect.NewRequest(&subscriptionv1pb.ListTrackedReposRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list tracked repos via grpc: %w", err)
	}

	repos := make([]monitoring.TrackedRepo, 0, len(resp.Msg.Repos))
	for _, r := range resp.Msg.Repos {
		tag, err := a.cursors.GetLastSeenTag(ctx, r.RepoId)
		if err != nil {
			return nil, err
		}
		repos = append(repos, monitoring.TrackedRepo{
			ID:          r.RepoId,
			FullName:    r.FullName,
			LastSeenTag: tag,
		})
	}

	return repos, nil
}

func (a *cursorCatalogAdapter) UpdateLastSeenTagAtomic(ctx context.Context, repoID int64, fullName, tag string, onTx func(context.Context, db.DBTX) error) error {
	return a.cursors.UpdateLastSeenTagAtomic(ctx, repoID, fullName, tag, onTx)
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
