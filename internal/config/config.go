package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	ErrMissingDatabaseURL     = errors.New("DATABASE_URL is required")
	ErrInvalidPortFormat      = errors.New("PORT must be a valid integer")
	ErrInvalidPort            = errors.New("PORT must be a valid TCP port (1-65535)")
	ErrMissingNotificationURL = errors.New("NOTIFICATION_URL is required")
	ErrInvalidNotificationURL = errors.New("NOTIFICATION_URL must be a valid absolute URL")
)

type Config struct {
	DatabaseURL     string
	Port            string
	GithubToken     string
	NotificationURL string
	ApiKey          string
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
	return &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		Port:            os.Getenv("PORT"),
		GithubToken:     os.Getenv("GITHUB_TOKEN"),
		NotificationURL: os.Getenv("NOTIFICATION_URL"),
		ApiKey:          os.Getenv("API_KEY"),
	}
}

func (cfg *Config) applyDefaults() {
	if cfg.Port == "" {
		cfg.Port = "8080"
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

	if cfg.NotificationURL == "" {
		return ErrMissingNotificationURL
	}

	parsed, err := url.ParseRequestURI(cfg.NotificationURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidNotificationURL
	}

	return nil
}
