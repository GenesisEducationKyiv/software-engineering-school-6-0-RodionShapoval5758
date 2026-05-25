//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"GithubReleaseNotificationAPI/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

var (
	mailpitBaseURL string
	smtpHost       string
	smtpPort       string
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	if err := os.Chdir(root); err != nil {
		log.Fatalf("chdir to repo root: %v", err)
	}

	ctx := context.Background()

	dsn := resolvePostgres(ctx)
	resolveMailpit(ctx)

	if err := db.RunMigrations(dsn); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("create db pool: %v", err)
	}
	testPool = pool
	defer testPool.Close()

	os.Exit(m.Run())
}

func resolvePostgres(ctx context.Context) string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}

	ctr, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("github_release_notifications_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("password"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("get postgres connection string: %v", err)
	}
	return dsn
}

func resolveMailpit(ctx context.Context) {
	if u := os.Getenv("TEST_MAILPIT_URL"); u != "" {
		mailpitBaseURL = u
		smtpHost = os.Getenv("TEST_SMTP_HOST")
		if smtpHost == "" {
			smtpHost = "localhost"
		}
		smtpPort = os.Getenv("TEST_SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "1025"
		}
		return
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "axllent/mailpit",
			ExposedPorts: []string{"1025/tcp", "8025/tcp"},
			Env: map[string]string{
				"MP_SMTP_AUTH_ACCEPT_ANY":     "1",
				"MP_SMTP_AUTH_ALLOW_INSECURE": "1",
			},
			WaitingFor: wait.ForListeningPort("8025/tcp"),
		},
		Started: true,
	})
	if err != nil {
		log.Fatalf("start mailpit container: %v", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		log.Fatalf("get mailpit host: %v", err)
	}
	httpPort, err := ctr.MappedPort(ctx, "8025/tcp")
	if err != nil {
		log.Fatalf("get mailpit HTTP port: %v", err)
	}
	smtpMapped, err := ctr.MappedPort(ctx, "1025/tcp")
	if err != nil {
		log.Fatalf("get mailpit SMTP port: %v", err)
	}

	mailpitBaseURL = fmt.Sprintf("http://%s:%s", host, httpPort.Port())
	smtpHost = host
	smtpPort = smtpMapped.Port()
}
