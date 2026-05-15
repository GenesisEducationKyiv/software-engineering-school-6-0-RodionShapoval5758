package config

import (
	"strings"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/app")
	t.Setenv("SMTP_HOST", "localhost")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.AppBaseURL != "http://localhost:8080" {
		t.Fatalf("AppBaseURL = %q, want %q", cfg.AppBaseURL, "http://localhost:8080")
	}
	if cfg.SMTPPort != "1025" {
		t.Fatalf("SMTPPort = %q, want %q", cfg.SMTPPort, "1025")
	}
	if cfg.FromEmail != "noreply@localhost" {
		t.Fatalf("FromEmail = %q, want %q", cfg.FromEmail, "noreply@localhost")
	}
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name: "missing database url",
			env: map[string]string{
				"SMTP_HOST": "localhost",
			},
			wantErr: "DATABASE_URL is required",
		},
		{
			name: "invalid port",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"PORT":         "bad-port",
			},
			wantErr: "PORT must be a valid TCP port",
		},
		{
			name: "invalid main url",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"MAIN_URL":     "://bad",
			},
			wantErr: "MAIN_URL must be a valid absolute URL",
		},
		{
			name: "missing smtp host",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
			},
			wantErr: "SMTP_HOST is required",
		},
		{
			name: "invalid smtp port",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"SMTP_PORT":    "99999",
			},
			wantErr: "SMTP_PORT must be a valid TCP port",
		},
		{
			name: "smtp auth user without password",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"SMTP_USER":    "user",
			},
			wantErr: "SMTP_USER and SMTP_PASSWORD must be configured together",
		},
		{
			name: "valid explicit config",
			env: map[string]string{
				"DATABASE_URL":  "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":     "localhost",
				"PORT":          "9090",
				"MAIN_URL":      "https://example.com",
				"SMTP_PORT":     "2525",
				"SMTP_USER":     "user",
				"SMTP_PASSWORD": "pass",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			_, err := Load()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Load() error = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("Load() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"DATABASE_URL",
		"PORT",
		"GITHUB_TOKEN",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USER",
		"SMTP_PASSWORD",
		"SENDER_EMAIL",
		"MAIN_URL",
		"API_KEY",
	} {
		t.Setenv(key, "")
	}
}
