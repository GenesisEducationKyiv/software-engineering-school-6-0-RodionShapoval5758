package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/monitoring/internal/catalogclient"
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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
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

	grpcTLSConfig, err := newGRPCClientTLSConfig(cfg)
	if err != nil {
		return fmt.Errorf("grpc tls config: %w", err)
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
		grpc.WithTransportCredentials(credentials.NewTLS(grpcTLSConfig)),
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
	defer func() { _ = conn.Close() }()

	grpcClient := catalogv1.NewCatalogServiceClient(conn)

	githubClient := github.NewGithubClient(&http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
		},
	}, &cfg.GithubToken)

	catalogAdapter := catalogclient.New(cursorStore, grpcClient)
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
