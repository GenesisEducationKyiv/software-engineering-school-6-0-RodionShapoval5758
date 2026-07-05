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
	ErrInvalidGRPCPortFmt   = errors.New("GRPC_PORT must be a valid integer")
	ErrInvalidGRPCPort      = errors.New("GRPC_PORT must be a valid TCP port (1-65535)")
	ErrMissingGRPCTLSCACert = errors.New("GRPC_TLS_CA_CERT is required")
	ErrMissingGRPCTLSCert   = errors.New("GRPC_TLS_CERT is required")
	ErrMissingGRPCTLSKey    = errors.New("GRPC_TLS_KEY is required")
	ErrMissingJWTPrivateKey = errors.New("JWT_PRIVATE_KEY is required")
)

type Config struct {
	DatabaseURL     string
	Port            string
	GRPCPort        string
	NATSUrl         string
	GRPCTLSCACert   string
	GRPCTLSCert     string
	GRPCTLSKey      string
	JWTPrivateKey   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	VerifyTokenTTL  time.Duration
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
		GRPCPort:        os.Getenv("GRPC_PORT"),
		NATSUrl:         os.Getenv("NATS_URL"),
		GRPCTLSCACert:   os.Getenv("GRPC_TLS_CA_CERT"),
		GRPCTLSCert:     os.Getenv("GRPC_TLS_CERT"),
		GRPCTLSKey:      os.Getenv("GRPC_TLS_KEY"),
		JWTPrivateKey:   os.Getenv("JWT_PRIVATE_KEY"),
		AccessTokenTTL:  durationFromEnv("JWT_ACCESS_TTL"),
		RefreshTokenTTL: durationFromEnv("JWT_REFRESH_TTL"),
		VerifyTokenTTL:  durationFromEnv("VERIFY_TOKEN_TTL"),
	}
}

func durationFromEnv(key string) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return 0
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0
	}

	return parsed
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

	if cfg.AccessTokenTTL == 0 {
		cfg.AccessTokenTTL = 15 * time.Minute
	}

	if cfg.RefreshTokenTTL == 0 {
		cfg.RefreshTokenTTL = 30 * 24 * time.Hour
	}

	if cfg.VerifyTokenTTL == 0 {
		cfg.VerifyTokenTTL = 24 * time.Hour
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
	if cfg.JWTPrivateKey == "" {
		return ErrMissingJWTPrivateKey
	}

	return nil
}
