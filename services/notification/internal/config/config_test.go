package config

import (
	"testing"
)

func validEnv() map[string]string {
	return map[string]string{
		"NATS_URL":      "nats://localhost:4222",
		"SMTP_HOST":     "localhost",
		"SMTP_PORT":     "1025",
		"SMTP_USER":     "",
		"SMTP_PASSWORD": "",
		"SENDER_EMAIL":  "noreply@localhost",
		"MAIN_URL":      "http://localhost:8080",
	}
}

func buildConfig(env map[string]string) *Config {
	return &Config{
		NATSUrl:    env["NATS_URL"],
		SMTPHost:   env["SMTP_HOST"],
		SMTPPort:   env["SMTP_PORT"],
		SMTPUser:   env["SMTP_USER"],
		SMTPPass:   env["SMTP_PASSWORD"],
		FromEmail:  env["SENDER_EMAIL"],
		AppBaseURL: env["MAIN_URL"],
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr error
	}{
		{
			name:    "valid minimal config",
			mutate:  nil,
			wantErr: nil,
		},
		{
			name:    "missing smtp host",
			mutate:  func(c *Config) { c.SMTPHost = "" },
			wantErr: ErrMissingSMTPHost,
		},
		{
			name:    "invalid smtp port format",
			mutate:  func(c *Config) { c.SMTPPort = "abc" },
			wantErr: ErrInvalidSMTPPortFormat,
		},
		{
			name:    "smtp port out of range",
			mutate:  func(c *Config) { c.SMTPPort = "99999" },
			wantErr: ErrInvalidSMTPPort,
		},
		{
			name:    "smtp port zero",
			mutate:  func(c *Config) { c.SMTPPort = "0" },
			wantErr: ErrInvalidSMTPPort,
		},
		{
			name:    "smtp user without password",
			mutate:  func(c *Config) { c.SMTPUser = "user" },
			wantErr: ErrInvalidSMTPCredentials,
		},
		{
			name:    "smtp password without user",
			mutate:  func(c *Config) { c.SMTPPass = "pass" },
			wantErr: ErrInvalidSMTPCredentials,
		},
		{
			name:    "missing app base url",
			mutate:  func(c *Config) { c.AppBaseURL = "" },
			wantErr: ErrMissingAppBaseURL,
		},
		{
			name:    "invalid app base url",
			mutate:  func(c *Config) { c.AppBaseURL = "not-a-url" },
			wantErr: ErrInvalidAppBaseURL,
		},
		{
			name:    "valid with credentials",
			mutate:  func(c *Config) { c.SMTPUser = "u"; c.SMTPPass = "p" },
			wantErr: nil,
		},
		{
			name:    "valid with app base url",
			mutate:  func(c *Config) { c.AppBaseURL = "http://example.com" },
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := buildConfig(validEnv())
			if tc.mutate != nil {
				tc.mutate(cfg)
			}

			err := cfg.validate()
			if err != tc.wantErr {
				t.Errorf("validate() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if cfg.NATSUrl != "nats://localhost:4222" {
		t.Errorf("NATSUrl default wrong: %s", cfg.NATSUrl)
	}
	if cfg.SMTPPort != "1025" {
		t.Errorf("SMTPPort default wrong: %s", cfg.SMTPPort)
	}
	if cfg.FromEmail != "noreply@localhost" {
		t.Errorf("FromEmail default wrong: %s", cfg.FromEmail)
	}
}
