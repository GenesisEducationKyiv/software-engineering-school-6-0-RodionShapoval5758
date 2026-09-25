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

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/notification/internal/config"
	"GithubReleaseNotificationAPI/services/notification/internal/consumer"
	"GithubReleaseNotificationAPI/services/notification/internal/health"
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

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", health.Handler(map[string]health.Pinger{
		"nats": &natsPinger{nc: nc},
	}))
	healthSrv := &http.Server{Addr: ":" + cfg.Port, Handler: healthMux}

	slog.Info("notification service started", "port", cfg.Port)

	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health server error", "error", err)
		}
	}()

	go func() {
		if err := consumer.StartDLQInspector(ctx, js); err != nil {
			slog.Error("dlq inspector failed", "error", err)
		}
	}()

	consumerErr := c.Start(ctx)

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}

	if consumerErr != nil {
		return consumerErr
	}

	if err := nc.Drain(); err != nil {
		slog.Error("nats drain failed", "error", err)
	}

	slog.Info("shutdown complete")

	return nil
}

type natsPinger struct{ nc *nats.Conn }

func (n *natsPinger) Ping(_ context.Context) error {
	if n.nc.Status() != nats.CONNECTED {
		return errors.New("not connected")
	}
	return nil
}
