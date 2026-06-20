package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/notification/internal/config"
	"GithubReleaseNotificationAPI/services/notification/internal/consumer"
	"GithubReleaseNotificationAPI/services/notification/internal/mailer"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With(slog.Group("service", slog.String("name", "notification-service")))
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

	nc, err := nats.Connect(cfg.NATSUrl, nats.Name("notification-service"))
	if err != nil {
		return err
	}
	defer nc.Drain()

	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       contract.StreamName,
		Subjects:   []string{contract.SubjectAll},
		Storage:    jetstream.FileStorage,
		Retention:  jetstream.WorkQueuePolicy,
		Duplicates: 2 * time.Minute,
		MaxAge:     24 * time.Hour,
		MaxBytes:   512 * 1024 * 1024,
	})
	if err != nil {
		return err
	}

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       contract.StreamDLQ,
		Subjects:   []string{contract.SubjectDead},
		Storage:    jetstream.FileStorage,
		Duplicates: 2 * time.Minute,
		MaxAge:     7 * 24 * time.Hour,
		MaxMsgs:    10_000,
	})
	if err != nil {
		return err
	}

	m := mailer.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail, cfg.AppBaseURL)
	c := consumer.New(js, m)

	slog.Info("notification service started")

	go func() {
		if err := consumer.StartDLQInspector(ctx, js); err != nil {
			slog.Error("dlq inspector failed", "error", err)
		}
	}()

	if err := c.Start(ctx); err != nil {
		return err
	}

	slog.Info("shutdown complete")
	return nil
}
