package main

import (
	"GithubReleaseNotificationAPI/internal/config"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/github"
	httpHandler "GithubReleaseNotificationAPI/internal/http/handler"
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/mail"
	"GithubReleaseNotificationAPI/internal/notifier"
	"GithubReleaseNotificationAPI/internal/service"
	"GithubReleaseNotificationAPI/internal/store/repository"
	"GithubReleaseNotificationAPI/internal/store/subscription"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load env variables", "error", err)
		os.Exit(1)
	}

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Pool creation error", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	subscriptionRepository := subscription.NewSubscriptionRepository(dbPool)
	repositoryRepository := repository.NewRepositoryRepository(dbPool)

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

	worker := notifier.NewWorker(
		smtpClient,
		githubClient,
		subscriptionRepository,
		repositoryRepository,
	)

	go func() {
		err := worker.Start(shutdownSignalCtx, time.Second*25)
		if err != nil {
			slog.Error("worker failed", "error", err)
		}
	}()

	<-shutdownSignalCtx.Done()
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("http server stopped")
}
