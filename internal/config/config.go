package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	GithubToken string

	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	FromEmail  string
	AppBaseURL string

	ApiKey string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		GithubToken: os.Getenv("GITHUB_TOKEN"),
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    os.Getenv("SMTP_PORT"),
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASSWORD"),
		FromEmail:   os.Getenv("SENDER_EMAIL"),
		AppBaseURL:  os.Getenv("MAIN_URL"),
		ApiKey: 	 os.Getenv("API_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.AppBaseURL == "" {
		cfg.AppBaseURL = "http://localhost:" + cfg.Port
	}

	if cfg.SMTPHost == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "1025"
	}
	if cfg.FromEmail == "" {
		cfg.FromEmail = "noreply@localhost"
	}

	return cfg, nil
}
