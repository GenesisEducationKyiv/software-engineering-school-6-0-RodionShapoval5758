package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/notification/internal/config"
	"GithubReleaseNotificationAPI/services/notification/internal/consumer"
	"GithubReleaseNotificationAPI/services/notification/internal/mailer"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})).With(slog.Group("service", slog.String("name", "notification-service")))
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
		Name:     contract.StreamName,
		Subjects: []string{contract.SubjectAll},
	})
	if err != nil {
		return err
	}

	m := mailer.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail, cfg.AppBaseURL)
	c := consumer.New(js, m)

	slog.Info("notification service started")

	if err := c.Start(ctx); err != nil {
		return err
	}

	slog.Info("shutdown complete")
	return nil
}
