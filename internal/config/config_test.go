package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAppliesDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/app")
	t.Setenv("SMTP_HOST", "localhost")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "http://localhost:8080", cfg.AppBaseURL)
	assert.Equal(t, "1025", cfg.SMTPPort)
	assert.Equal(t, "noreply@localhost", cfg.FromEmail)
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr error
	}{
		{
			name: "missing database url",
			env: map[string]string{
				"SMTP_HOST": "localhost",
			},
			wantErr: ErrMissingDatabaseURL,
		},
		{
			name: "invalid port",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"PORT":         "bad-port",
			},
			wantErr: ErrInvalidPort,
		},
		{
			name: "invalid main url",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"MAIN_URL":     "://bad",
			},
			wantErr: ErrInvalidMainURL,
		},
		{
			name: "missing smtp host",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
			},
			wantErr: ErrMissingSMTPHost,
		},
		{
			name: "invalid smtp port",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"SMTP_PORT":    "99999",
			},
			wantErr: ErrInvalidSMTPPort,
		},
		{
			name: "smtp auth user without password",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"SMTP_HOST":    "localhost",
				"SMTP_USER":    "user",
			},
			wantErr: ErrInvalidSMTPCredentials,
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
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.wantErr)
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
		original, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q): %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				if err := os.Setenv(key, original); err != nil {
					t.Errorf("os.Setenv(%q): %v", key, err)
				}
			} else {
				if err := os.Unsetenv(key); err != nil {
					t.Errorf("os.Unsetenv(%q): %v", key, err)
				}
			}
		})
	}
}
