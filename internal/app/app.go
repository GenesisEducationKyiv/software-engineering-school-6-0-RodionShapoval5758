package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/config"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/fanout"
	"GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/outbox"
	"GithubReleaseNotificationAPI/internal/subscription"
	"GithubReleaseNotificationAPI/internal/subscription/usecase"
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
	fanout     *fanout.Worker
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

	if _, err := js.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:       contract.StreamName,
		Subjects:   []string{contract.SubjectAll},
		Storage:    jetstream.FileStorage,
		Retention:  jetstream.WorkQueuePolicy,
		Duplicates: 2 * time.Minute,
		MaxAge:     24 * time.Hour,
		MaxBytes:   512 * 1024 * 1024,
	}); err != nil {
		_ = nc.Drain()
		dbPool.Close()
		return nil, fmt.Errorf("ensure notification stream: %w", err)
	}

	outboxStore := outbox.NewStore()
	fanoutStore := fanout.NewStore()
	outboxRelay := outbox.NewRelay(dbPool, js, outboxStore)

	subRepo := subscription.NewRepository(dbPool)
	githubClient := github.NewGithubClient(&http.Client{Timeout: 15 * time.Second}, &cfg.GithubToken)

	ensureUC := catalog.NewEnsure(dbPool)
	deleteIfOrphanedUC := catalog.NewDeleteIfOrphaned(dbPool)
	listTrackedUC := catalog.NewListTracked(dbPool)
	updateReleaseUC := catalog.NewUpdateLastSeenTagAtomic(dbPool)

	subscribeUC := usecase.NewSubscribe(subRepo, ensureUC, githubClient, outboxStore, db.WrapPool(dbPool))
	confirmUC := usecase.NewConfirm(subRepo)
	unsubscribeUC := usecase.NewUnsubscribe(subRepo, deleteIfOrphanedUC)
	listUC := usecase.NewList(subRepo)

	reg := prometheus.NewRegistry()
	appMetrics := metrics.New(reg)

	subHandler := handler.New(subscribeUC, confirmUC, unsubscribeUC, listUC)
	router := httpRouter.New(subHandler, cfg.ApiKey, appMetrics, &dbPinger{dbPool}, &natsPinger{nc})

	catalogAdapter := &catalogMonitoringAdapter{lister: listTrackedUC, updater: updateReleaseUC}
	worker := monitoring.NewWorker(githubClient, catalogAdapter, fanoutStore, appMetrics)
	fanoutWorker := fanout.NewWorker(dbPool, &recipientListerAdapter{lister: listUC}, outboxStore, fanoutStore)

	return &App{
		server:     &http.Server{Addr: ":" + cfg.Port, Handler: router},
		worker:     worker,
		relay:      outboxRelay,
		fanout:     fanoutWorker,
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
	go a.fanout.Run(ctx)
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
