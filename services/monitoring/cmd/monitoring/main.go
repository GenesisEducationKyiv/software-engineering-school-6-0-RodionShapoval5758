package main

import (
	"context"
	"encoding/json"
	"errors"
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
	"GithubReleaseNotificationAPI/services/monitoring/internal/health"
	"GithubReleaseNotificationAPI/services/monitoring/internal/monitoring"
	monrelay "GithubReleaseNotificationAPI/services/monitoring/internal/relay"
	"GithubReleaseNotificationAPI/services/monitoring/internal/store"
	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"

	"github.com/jackc/pgx/v5/pgxpool"
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

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", health.Handler(map[string]health.Pinger{
		"db":   &dbPinger{pool: pool},
		"nats": &natsPinger{nc: nc},
	}))
	healthSrv := &http.Server{Addr: ":" + cfg.Port, Handler: healthMux}

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

	slog.Info("monitoring service started", "scan_interval", cfg.ScanInterval, "port", cfg.Port)

	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health server error", "error", err)
		}
	}()
	go relay.Run(ctx)

	workerErr := worker.Start(ctx, cfg.ScanInterval)

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}

	if workerErr != nil {
		return workerErr
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
