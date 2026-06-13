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
	t.Setenv("NOTIFICATION_URL", "http://notification:8081")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr error
	}{
		{
			name:    "missing database url",
			env:     map[string]string{},
			wantErr: ErrMissingDatabaseURL,
		},
		{
			name: "invalid port format",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"PORT":         "bad-port",
			},
			wantErr: ErrInvalidPortFormat,
		},
		{
			name: "invalid port range",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
				"PORT":         "99999",
			},
			wantErr: ErrInvalidPort,
		},
		{
			name: "missing notification url",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/app",
			},
			wantErr: ErrMissingNotificationURL,
		},
		{
			name: "invalid notification url",
			env: map[string]string{
				"DATABASE_URL":     "postgres://user:pass@localhost:5432/app",
				"NOTIFICATION_URL": "://bad",
			},
			wantErr: ErrInvalidNotificationURL,
		},
		{
			name: "valid explicit config",
			env: map[string]string{
				"DATABASE_URL":     "postgres://user:pass@localhost:5432/app",
				"PORT":             "9090",
				"NOTIFICATION_URL": "http://notification:8081",
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
		"NOTIFICATION_URL",
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
