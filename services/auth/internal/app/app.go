package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"GithubReleaseNotificationAPI/contract"
	authv1 "GithubReleaseNotificationAPI/services/auth/api/gen/authv1/auth/v1"
	"GithubReleaseNotificationAPI/services/auth/internal/config"
	"GithubReleaseNotificationAPI/services/auth/internal/db"
	"GithubReleaseNotificationAPI/services/auth/internal/outbox"
	"GithubReleaseNotificationAPI/services/auth/internal/token"
	grpchandler "GithubReleaseNotificationAPI/services/auth/internal/transport/grpc/handler"
	httphandler "GithubReleaseNotificationAPI/services/auth/internal/transport/http/handler"
	httprouter "GithubReleaseNotificationAPI/services/auth/internal/transport/http/router"
	"GithubReleaseNotificationAPI/services/auth/internal/user"
	"GithubReleaseNotificationAPI/services/auth/internal/user/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type App struct {
	httpServer *http.Server
	grpcServer *grpc.Server
	relay      *outbox.Relay
	dbPool     *pgxpool.Pool
	nc         *natsgo.Conn
}

func Build(cfg *config.Config) (*App, error) {
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grpcTLSConfig, err := newGRPCServerTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("grpc tls config: %w", err)
	}

	signer, err := token.LoadSigner(cfg.JWTPrivateKey, cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		return nil, err
	}

	dbPool, err := db.NewPool(initCtx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	nc, err := natsgo.Connect(cfg.NATSUrl, natsgo.Name("auth-service"))
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

	// Stream config must stay identical to the one the subscription service
	// ensures; CreateOrUpdateStream silently rewrites diverging configs.
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
	outboxRelay := outbox.NewRelay(dbPool, js, outboxStore)

	userStore := user.NewStore()
	txer := db.WrapPool(dbPool)

	registerUC := usecase.NewRegister(userStore, outboxStore, txer, cfg.VerifyTokenTTL)
	verifyUC := usecase.NewVerifyEmail(userStore, txer)
	loginUC := usecase.NewLogin(userStore, signer, dbPool, cfg.RefreshTokenTTL)
	refreshUC := usecase.NewRefresh(userStore, signer, txer, cfg.RefreshTokenTTL)
	logoutUC := usecase.NewLogout(userStore, dbPool)

	httpHandler := httphandler.New(registerUC, verifyUC, loginUC, refreshUC, logoutUC)
	chiRouter := httprouter.New(httpHandler, &dbPinger{dbPool}, &natsPinger{nc})

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(grpcTLSConfig)))
	authv1.RegisterAuthServiceServer(grpcServer, grpchandler.NewSigningKey(signer))

	return &App{
		httpServer: &http.Server{Addr: ":" + cfg.Port, Handler: chiRouter},
		grpcServer: grpcServer,
		relay:      outboxRelay,
		dbPool:     dbPool,
		nc:         nc,
	}, nil
}

func (a *App) Serve(ctx context.Context, grpcPort string) error {
	defer a.dbPool.Close()
	defer func() { _ = a.nc.Drain() }()

	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	slog.Info("starting HTTP server", "port", a.httpServer.Addr)
	slog.Info("starting gRPC server", "port", grpcPort)

	serverErr := make(chan error, 2)

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	go func() {
		if err := a.grpcServer.Serve(grpcLis); err != nil {
			serverErr <- err
		}
	}()

	go a.relay.Run(ctx)

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	a.grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("servers stopped")

	return nil
}
