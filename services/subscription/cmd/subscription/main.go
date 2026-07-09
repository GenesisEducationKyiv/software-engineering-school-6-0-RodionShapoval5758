package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"GithubReleaseNotificationAPI/services/subscription/internal/app"
	"GithubReleaseNotificationAPI/services/subscription/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With(slog.Group("service", slog.String("name", "subscription-service")))
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

	application, err := app.Build(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return application.Serve(ctx, cfg.GRPCPort)
}
