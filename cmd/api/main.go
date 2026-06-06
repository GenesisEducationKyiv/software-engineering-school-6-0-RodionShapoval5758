package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/config"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/github"
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/notification"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/prometheus/client_golang/prometheus"
)

type confirmedSubReader struct{ svc *subscription.Service }

func (r confirmedSubReader) ListConfirmedByRepositoryID(ctx context.Context, id int64) ([]monitoring.ConfirmedSubscriber, error) {
	subs, err := r.svc.ListConfirmedByRepositoryID(ctx, id)
	if err != nil {
		return nil, err
	}

	cs := make([]monitoring.ConfirmedSubscriber, len(subs))
	for i, s := range subs {
		cs[i] = monitoring.ConfirmedSubscriber{
			Email:            s.Email,
			UnsubscribeToken: s.UnsubscribeToken,
		}
	}

	return cs, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})).With(
		slog.Group("service", slog.String("name", "github-release-notification-api")))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		return err
	}

	initCtx, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelInit()

	dbPool, err := db.NewPool(initCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	catalogService := catalog.New(dbPool)
	subRepo := subscription.NewRepository(dbPool)

	githubClient := github.NewGithubClient(http.DefaultClient, &cfg.GithubToken)

	mailer := notification.NewMailer(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.FromEmail,
		cfg.AppBaseURL,
	)

	subService := subscription.NewService(subRepo, catalogService, githubClient, mailer)
	subHandler := subscription.New(subService)

	reg := prometheus.NewRegistry()
	appMetrics := metrics.New(reg)
	router := httpRouter.New(subHandler, cfg.ApiKey, appMetrics)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	slog.Info("starting HTTP server", "port", cfg.Port)

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	shutdownSignalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go appMetrics.CollectDBStats(shutdownSignalCtx, dbPool, 15*time.Second)

	releaseNotifier := monitoring.NewReleaseNotifier(mailer, confirmedSubReader{svc: subService})
	worker := monitoring.NewWorker(githubClient, catalogService, releaseNotifier, appMetrics)

	go func() {
		if err := worker.Start(shutdownSignalCtx, 25*time.Second); err != nil {
			slog.Error("worker failed", "error", err)
		}
	}()

	select {
	case <-shutdownSignalCtx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("http server stopped")

	return nil
}
