package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrMissingDatabaseURL          = errors.New("DATABASE_URL is required")
	ErrMissingNATSUrl              = errors.New("NATS_URL is required")
	ErrMissingSubscriptionGRPCAddr = errors.New("SUBSCRIPTION_GRPC_ADDR is required")
	ErrMissingGRPCTLSCACert        = errors.New("GRPC_TLS_CA_CERT is required")
	ErrMissingGRPCTLSCert          = errors.New("GRPC_TLS_CERT is required")
	ErrMissingGRPCTLSKey           = errors.New("GRPC_TLS_KEY is required")
)

type Config struct {
	DatabaseURL          string
	GithubToken          string
	NATSUrl              string
	ScanInterval         time.Duration
	SubscriptionGRPCAddr string
	GRPCTLSCACert        string
	GRPCTLSCert          string
	GRPCTLSKey           string
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
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		GithubToken:          os.Getenv("GITHUB_TOKEN"),
		NATSUrl:              os.Getenv("NATS_URL"),
		ScanInterval:         interval,
		SubscriptionGRPCAddr: os.Getenv("SUBSCRIPTION_GRPC_ADDR"),
		GRPCTLSCACert:        os.Getenv("GRPC_TLS_CA_CERT"),
		GRPCTLSCert:          os.Getenv("GRPC_TLS_CERT"),
		GRPCTLSKey:           os.Getenv("GRPC_TLS_KEY"),
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
	if c.SubscriptionGRPCAddr == "" {
		return ErrMissingSubscriptionGRPCAddr
	}
	if c.GRPCTLSCACert == "" {
		return ErrMissingGRPCTLSCACert
	}
	if c.GRPCTLSCert == "" {
		return ErrMissingGRPCTLSCert
	}
	if c.GRPCTLSKey == "" {
		return ErrMissingGRPCTLSKey
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
