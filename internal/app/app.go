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
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/notifier"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
)

type App struct {
	server     *http.Server
	worker     *monitoring.Worker
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

	publisher := notifier.NewPublisher(js)

	catalogService := catalog.New(dbPool)
	subRepo := subscription.NewRepository(dbPool)
	githubClient := github.NewGithubClient(http.DefaultClient, &cfg.GithubToken)
	subService := subscription.NewService(subRepo, catalogService, githubClient, publisher)

	reg := prometheus.NewRegistry()
	appMetrics := metrics.New(reg)
	router := httpRouter.New(subscription.New(subService), cfg.ApiKey, appMetrics)

	releaseNotifier := monitoring.NewReleaseNotifier(publisher, NewConfirmedSubReader(subService))
	worker := monitoring.NewWorker(githubClient, catalogService, releaseNotifier, appMetrics)

	return &App{
		server:     &http.Server{Addr: ":" + cfg.Port, Handler: router},
		worker:     worker,
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
