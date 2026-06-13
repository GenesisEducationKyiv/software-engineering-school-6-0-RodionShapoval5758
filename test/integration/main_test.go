//go:build integration

package integration_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"GithubReleaseNotificationAPI/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	if err := os.Chdir(root); err != nil {
		log.Fatalf("chdir to repo root: %v", err)
	}

	ctx := context.Background()

	dsn, pgCleanup := resolvePostgres(ctx)
	defer pgCleanup()

	if err := db.RunMigrations(dsn); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("create db pool: %v", err)
	}
	testPool = pool
	defer testPool.Close()

	return m.Run()
}

func resolvePostgres(ctx context.Context) (string, func()) {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn, func() {}
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

	return dsn, func() {
		if err := ctr.Terminate(ctx); err != nil {
			log.Printf("terminate postgres container: %v", err)
		}
	}
}
