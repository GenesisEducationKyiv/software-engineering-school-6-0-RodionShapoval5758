package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog"
	"GithubReleaseNotificationAPI/services/subscription/internal/config"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	"GithubReleaseNotificationAPI/services/subscription/internal/fanout"
	"GithubReleaseNotificationAPI/services/subscription/internal/github"
	"GithubReleaseNotificationAPI/services/subscription/internal/metrics"
	"GithubReleaseNotificationAPI/services/subscription/internal/outbox"
	"GithubReleaseNotificationAPI/services/subscription/internal/saga"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription/usecase"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/handler"
	httpRouter "GithubReleaseNotificationAPI/services/subscription/internal/transport/http/router"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
)

type App struct {
	server     *http.Server
	relay      *outbox.Relay
	fanout     *fanout.Worker
	sagaCons   *saga.Consumer
	sagaReaper *saga.Reaper
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

	if _, err := js.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:       contract.StreamTracking,
		Subjects:   []string{contract.SubjectTrackingAll},
		Storage:    jetstream.FileStorage,
		Duplicates: 2 * time.Minute,
		MaxAge:     24 * time.Hour,
	}); err != nil {
		_ = nc.Drain()
		dbPool.Close()
		return nil, fmt.Errorf("ensure tracking stream: %w", err)
	}

	if _, err := js.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:       contract.StreamSaga,
		Subjects:   []string{contract.SubjectSagaAll},
		Storage:    jetstream.FileStorage,
		Duplicates: 2 * time.Minute,
		MaxAge:     7 * 24 * time.Hour,
	}); err != nil {
		_ = nc.Drain()
		dbPool.Close()
		return nil, fmt.Errorf("ensure saga stream: %w", err)
	}

	outboxStore := outbox.NewStore()
	outboxRelay := outbox.NewRelay(dbPool, js, outboxStore)

	subRepo := subscription.NewRepository(dbPool)
	githubClient := github.NewGithubClient(&http.Client{Timeout: 15 * time.Second}, &cfg.GithubToken)

	ensureUC := catalog.NewEnsure(dbPool)
	deleteIfOrphanedUC := catalog.NewDeleteIfOrphaned(dbPool, outboxStore)

	sagaStore := saga.NewStore()
	sagaOrchestrator := saga.NewOrchestrator(sagaStore, db.WrapPool(dbPool), deleteIfOrphanedUC, subRepo)

	subscribeUC := usecase.NewSubscribe(subRepo, ensureUC, githubClient, outboxStore, sagaStore, db.WrapPool(dbPool), cfg.SagaConfirmTTL)
	confirmUC := usecase.NewConfirm(subRepo, sagaOrchestrator)
	unsubscribeUC := usecase.NewUnsubscribe(subRepo, deleteIfOrphanedUC)
	listUC := usecase.NewList(subRepo)

	reg := prometheus.NewRegistry()
	appMetrics := metrics.New(reg)

	subHandler := handler.New(subscribeUC, confirmUC, unsubscribeUC, listUC)
	router := httpRouter.New(subHandler, cfg.ApiKey, appMetrics, &dbPinger{dbPool}, &natsPinger{nc})

	fanoutWorker := fanout.NewWorker(js, dbPool, &recipientListerAdapter{lister: listUC}, outboxStore)
	sagaConsumer := saga.NewConsumer(js, sagaOrchestrator)
	sagaReaper := saga.NewReaper(dbPool, sagaStore, sagaOrchestrator)

	return &App{
		server:     &http.Server{Addr: ":" + cfg.Port, Handler: router},
		relay:      outboxRelay,
		fanout:     fanoutWorker,
		sagaCons:   sagaConsumer,
		sagaReaper: sagaReaper,
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
		if err := a.fanout.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("fanout consumer error", "error", err)
		}
	}()
	go func() {
		if err := a.sagaCons.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("saga consumer error", "error", err)
		}
	}()
	go a.sagaReaper.Run(ctx)

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
