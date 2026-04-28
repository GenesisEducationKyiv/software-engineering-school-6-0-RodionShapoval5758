package db

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn string) error {
	mgrt, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func(mgrt *migrate.Migrate) {
		sourceErr, databaseErr := mgrt.Close()
		if sourceErr != nil {
			slog.Warn("failed to close migration source", "error", sourceErr)
		}
		if databaseErr != nil {
			slog.Warn("failed to close migration database", "error", databaseErr)
		}
	}(mgrt)

	if err := mgrt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("migrations ran successfully")
	return nil
}
