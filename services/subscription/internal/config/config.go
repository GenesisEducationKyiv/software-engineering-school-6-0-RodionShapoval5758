package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrMissingDatabaseURL = errors.New("DATABASE_URL is required")
	ErrInvalidPortFormat  = errors.New("PORT must be a valid integer")
	ErrInvalidPort        = errors.New("PORT must be a valid TCP port (1-65535)")
)

type Config struct {
	DatabaseURL    string
	Port           string
	GithubToken    string
	NATSUrl        string
	ApiKey         string
	SagaConfirmTTL time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := loadFromEnv()
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadFromEnv() *Config {
	ttl := 24 * time.Hour
	if raw := os.Getenv("SAGA_CONFIRM_TTL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			ttl = parsed
		}
	}

	return &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		Port:           os.Getenv("PORT"),
		GithubToken:    os.Getenv("GITHUB_TOKEN"),
		NATSUrl:        os.Getenv("NATS_URL"),
		ApiKey:         os.Getenv("API_KEY"),
		SagaConfirmTTL: ttl,
	}
}

func (cfg *Config) applyDefaults() {
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.NATSUrl == "" {
		cfg.NATSUrl = "nats://localhost:4222"
	}
}

func (cfg *Config) validate() error {
	if cfg.DatabaseURL == "" {
		return ErrMissingDatabaseURL
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return ErrInvalidPortFormat
	}
	if port <= 0 || port > 65535 {
		return ErrInvalidPort
	}

	return nil
}
