package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/config"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/notifier"
	"GithubReleaseNotificationAPI/internal/outbox"
	"GithubReleaseNotificationAPI/internal/subscription"
	"GithubReleaseNotificationAPI/internal/transport/http/handler"
	httpRouter "GithubReleaseNotificationAPI/internal/transport/http/router"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
)

type App struct {
	server     *http.Server
	worker     *monitoring.Worker
	relay      *outbox.Relay
	appMetrics *metrics.Metrics
	dbPool     *pgxpool.Pool
	nc         *natsgo.Conn
}

func Build(cfg *config.Config) (*App, error) {
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		return nil, err
	}

	dbPool, err := db.NewPool(initCtx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	nc, err := natsgo.Connect(cfg.NATSUrl, natsgo.Name("github-release-notification-api"))
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		_ = nc.Drain()
		dbPool.Close()
		return nil, fmt.Errorf("init jetstream: %w", err)
	}

	if err := notifier.EnsureStream(initCtx, js); err != nil {
		_ = nc.Drain()
		dbPool.Close()
		return nil, fmt.Errorf("ensure notification stream: %w", err)
	}

	outboxRelay := outbox.NewRelay(dbPool, js)
	outboxStore := &outboxStoreAdapter{}

	catalogService := catalog.New(dbPool)
	subRepo := subscription.NewRepository(dbPool)
	githubClient := github.NewGithubClient(http.DefaultClient, &cfg.GithubToken)
	subService := subscription.NewService(subRepo, catalogService, githubClient, outboxStore, db.WrapPool(dbPool))

	reg := prometheus.NewRegistry()
	appMetrics := metrics.New(reg)

	subHandler := handler.New(subService)
	internalHandler := handler.NewInternal(subService, cfg.InternalToken)
	router := httpRouter.New(subHandler, internalHandler, cfg.ApiKey, appMetrics, dbPool, &natsPinger{nc})

	fanoutEnqueuer := &fanoutEnqueuerAdapter{}
	worker := monitoring.NewWorker(githubClient, catalogService, fanoutEnqueuer, appMetrics)

	return &App{
		server:     &http.Server{Addr: ":" + cfg.Port, Handler: router},
		worker:     worker,
		relay:      outboxRelay,
		appMetrics: appMetrics,
		dbPool:     dbPool,
		nc:         nc,
	}, nil
}

func (a *App) Serve(ctx context.Context) error {
	defer a.dbPool.Close()
	defer func() { _ = a.nc.Drain() }()

	slog.Info("starting HTTP server", "port", a.server.Addr)

	serverErr := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	go a.appMetrics.CollectDBStats(ctx, a.dbPool, 15*time.Second)
	go a.relay.Run(ctx)
	go func() {
		if err := a.worker.Start(ctx, 25*time.Second); err != nil {
			slog.Error("worker failed", "error", err)
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("http server stopped")

	return nil
}
