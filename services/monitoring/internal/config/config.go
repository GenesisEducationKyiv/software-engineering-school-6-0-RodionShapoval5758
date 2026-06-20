package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrMissingDatabaseURL = errors.New("DATABASE_URL is required")
	ErrMissingNATSUrl     = errors.New("NATS_URL is required")
)

type Config struct {
	DatabaseURL  string
	GithubToken  string
	NATSUrl      string
	ScanInterval time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := loadFromEnv()
	cfg.applyDefaults()
	return cfg, cfg.validate()
}

func loadFromEnv() *Config {
	interval, err := time.ParseDuration(os.Getenv("SCAN_INTERVAL"))
	if err != nil {
		interval = 0
	}
	return &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		GithubToken:  os.Getenv("GITHUB_TOKEN"),
		NATSUrl:      os.Getenv("NATS_URL"),
		ScanInterval: interval,
	}
}

func (c *Config) applyDefaults() {
	if c.ScanInterval == 0 {
		c.ScanInterval = 25 * time.Second
	}
	if c.NATSUrl == "" {
		c.NATSUrl = "nats://localhost:4222"
	}
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return ErrMissingDatabaseURL
	}
	return nil
}

// MigrationDSN appends a custom schema_migrations table name so monitoring
// migrations don't collide with the main app's schema_migrations tracking.
func (c *Config) MigrationDSN() string {
	sep := "?"
	if strings.Contains(c.DatabaseURL, "?") {
		sep = "&"
	}
	return c.DatabaseURL + sep + "x-migrations-table=monitoring_schema_migrations"
}
