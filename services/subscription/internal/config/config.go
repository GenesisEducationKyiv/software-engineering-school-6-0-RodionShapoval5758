package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrMissingDatabaseURL   = errors.New("DATABASE_URL is required")
	ErrInvalidPortFormat    = errors.New("PORT must be a valid integer")
	ErrInvalidPort          = errors.New("PORT must be a valid TCP port (1-65535)")
	ErrInvalidGRPCPort      = errors.New("GRPC_PORT must be a valid TCP port (1-65535)")
	ErrInvalidGRPCPortFmt   = errors.New("GRPC_PORT must be a valid integer")
	ErrMissingGRPCTLSCACert = errors.New("GRPC_TLS_CA_CERT is required")
	ErrMissingGRPCTLSCert   = errors.New("GRPC_TLS_CERT is required")
	ErrMissingGRPCTLSKey    = errors.New("GRPC_TLS_KEY is required")
	ErrMissingAuthGRPCAddr  = errors.New("AUTH_GRPC_ADDR is required")
)

type Config struct {
	DatabaseURL    string
	Port           string
	GRPCPort       string
	GithubToken    string
	NATSUrl        string
	AuthGRPCAddr   string
	SagaConfirmTTL time.Duration
	GRPCTLSCACert  string
	GRPCTLSCert    string
	GRPCTLSKey     string
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
		GRPCPort:       os.Getenv("GRPC_PORT"),
		GithubToken:    os.Getenv("GITHUB_TOKEN"),
		NATSUrl:        os.Getenv("NATS_URL"),
		AuthGRPCAddr:   os.Getenv("AUTH_GRPC_ADDR"),
		SagaConfirmTTL: ttl,
		GRPCTLSCACert:  os.Getenv("GRPC_TLS_CA_CERT"),
		GRPCTLSCert:    os.Getenv("GRPC_TLS_CERT"),
		GRPCTLSKey:     os.Getenv("GRPC_TLS_KEY"),
	}
}

func (cfg *Config) applyDefaults() {
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.GRPCPort == "" {
		cfg.GRPCPort = "50051"
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

	grpcPort, err := strconv.Atoi(cfg.GRPCPort)
	if err != nil {
		return ErrInvalidGRPCPortFmt
	}
	if grpcPort <= 0 || grpcPort > 65535 {
		return ErrInvalidGRPCPort
	}

	if cfg.GRPCTLSCACert == "" {
		return ErrMissingGRPCTLSCACert
	}
	if cfg.GRPCTLSCert == "" {
		return ErrMissingGRPCTLSCert
	}
	if cfg.GRPCTLSKey == "" {
		return ErrMissingGRPCTLSKey
	}
	if cfg.AuthGRPCAddr == "" {
		return ErrMissingAuthGRPCAddr
	}

	return nil
}
