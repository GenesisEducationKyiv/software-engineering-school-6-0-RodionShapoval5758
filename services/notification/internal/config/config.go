package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	ErrMissingSMTPHost        = errors.New("SMTP_HOST is required")
	ErrInvalidSMTPPortFormat  = errors.New("SMTP_PORT must be a valid integer")
	ErrInvalidSMTPPort        = errors.New("SMTP_PORT must be a valid TCP port (1-65535)")
	ErrInvalidSMTPCredentials = errors.New("SMTP_USER and SMTP_PASSWORD must be configured together")
	ErrMissingAppBaseURL      = errors.New("MAIN_URL is required")
	ErrInvalidAppBaseURL      = errors.New("MAIN_URL must be a valid absolute URL")
	ErrInvalidPortFormat      = errors.New("PORT must be a valid integer")
	ErrInvalidPort            = errors.New("PORT must be a valid TCP port (1-65535)")
)

type Config struct {
	Port       string
	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	FromEmail  string
	AppBaseURL string
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
		Port:       os.Getenv("PORT"),
		SMTPHost:   os.Getenv("SMTP_HOST"),
		SMTPPort:   os.Getenv("SMTP_PORT"),
		SMTPUser:   os.Getenv("SMTP_USER"),
		SMTPPass:   os.Getenv("SMTP_PASSWORD"),
		FromEmail:  os.Getenv("SENDER_EMAIL"),
		AppBaseURL: os.Getenv("MAIN_URL"),
	}
}

func (cfg *Config) applyDefaults() {
	if cfg.Port == "" {
		cfg.Port = "8081"
	}
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "1025"
	}
	if cfg.FromEmail == "" {
		cfg.FromEmail = "noreply@localhost"
	}
}

func (cfg *Config) validate() error {
	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return ErrInvalidPortFormat
	}
	if port <= 0 || port > 65535 {
		return ErrInvalidPort
	}

	if cfg.SMTPHost == "" {
		return ErrMissingSMTPHost
	}

	smtpPort, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil {
		return ErrInvalidSMTPPortFormat
	}
	if smtpPort <= 0 || smtpPort > 65535 {
		return ErrInvalidSMTPPort
	}

	if (cfg.SMTPUser == "") != (cfg.SMTPPass == "") {
		return ErrInvalidSMTPCredentials
	}

	if cfg.AppBaseURL == "" {
		return ErrMissingAppBaseURL
	}

	parsed, err := url.ParseRequestURI(cfg.AppBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidAppBaseURL
	}

	return nil
}
