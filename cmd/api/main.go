package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GithubReleaseNotificationAPI/internal/config"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/github"
	httpHandler "GithubReleaseNotificationAPI/internal/http/handler"
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/mail"
	"GithubReleaseNotificationAPI/internal/watcher"
	"GithubReleaseNotificationAPI/internal/service"
	"GithubReleaseNotificationAPI/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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

	subscriptionRepository := store.NewSubscriptionRepository(dbPool)
	repositoryRepository := store.NewRepoRepository(dbPool)

	githubClient := github.NewGithubClient(http.DefaultClient, &cfg.GithubToken)

	smtpClient := mail.NewSMTPService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.FromEmail,
		cfg.AppBaseURL,
	)

	subscriptionService := service.NewSubscriptionService(
		subscriptionRepository,
		repositoryRepository,
		githubClient,
		smtpClient,
	)

	handler := httpHandler.New(subscriptionService)
	router := httpRouter.New(handler, cfg.ApiKey)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	slog.Info("starting HTTP server", "port", cfg.Port)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdownSignalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	releaseNotifier := watcher.NewReleaseNotifier(smtpClient, subscriptionRepository)
	worker := watcher.NewWorker(githubClient, repositoryRepository, releaseNotifier)

	go func() {
		if err := worker.Start(shutdownSignalCtx, 25*time.Second); err != nil {
			slog.Error("worker failed", "error", err)
		}
	}()

	<-shutdownSignalCtx.Done()
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("http server stopped")

	return nil
}
