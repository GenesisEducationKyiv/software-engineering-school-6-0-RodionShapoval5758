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
	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	grpcListTrackedTimeout = 5 * time.Second
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

	conn, err := grpc.NewClient(
		cfg.SubscriptionGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultServiceConfig(`{
			"methodConfig": [{
				"name": [{"service": "catalog.v1.CatalogService"}],
				"retryPolicy": {
					"maxAttempts": 3,
					"initialBackoff": "0.5s",
					"maxBackoff": "5s",
					"backoffMultiplier": 2.0,
					"retryableStatusCodes": ["UNAVAILABLE"]
				}
			}]
		}`),
	)
	if err != nil {
		return fmt.Errorf("connect to subscription grpc: %w", err)
	}
	defer conn.Close()

	grpcClient := catalogv1.NewCatalogServiceClient(conn)

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
	grpc    catalogv1.CatalogServiceClient
}

func (a *cursorCatalogAdapter) ListTracked(ctx context.Context) ([]monitoring.TrackedRepo, error) {
	ctx, cancel := context.WithTimeout(ctx, grpcListTrackedTimeout)
	defer cancel()

	resp, err := a.grpc.ListTrackedRepos(ctx, &catalogv1.ListTrackedReposRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return nil, fmt.Errorf("list tracked repos via grpc: %w", err)
	}

	repos := make([]monitoring.TrackedRepo, 0, len(resp.Repos))
	for _, r := range resp.Repos {
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
