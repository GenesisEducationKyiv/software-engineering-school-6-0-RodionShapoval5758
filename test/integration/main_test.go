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
)

var testPool *pgxpool.Pool

const defaultTestDSN = "postgres://postgres:password@localhost:5432/github_release_notifications_test?sslmode=disable"

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	if err := os.Chdir(root); err != nil {
		log.Fatalf("chdir to repo root: %v", err)
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	if err := db.RunMigrations(dsn); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	pool, err := db.NewPool(context.Background(), dsn)
	if err != nil {
		log.Fatalf("create db pool: %v", err)
	}
	testPool = pool
	defer testPool.Close()

	os.Exit(m.Run())
}
