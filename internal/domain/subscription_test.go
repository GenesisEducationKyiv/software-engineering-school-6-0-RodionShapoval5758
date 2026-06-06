package domain

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 16", 16},
		{"length 32", 32},
		{"length 64", 64},
		{"zero length", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateToken(tt.length)

			require.NoError(t, err)

			if tt.length == 0 {
				assert.Empty(t, got)
				return
			}

			decoded, err := base64.RawURLEncoding.DecodeString(got)
			require.NoError(t, err, "result should be valid base64")
			require.Len(t, decoded, tt.length)

			got2, _ := generateToken(tt.length)
			assert.NotEqual(t, got, got2, "should not produce the same token twice")
		})
	}
}

func TestNewSubscription(t *testing.T) {
	email := "test@example.com"
	repoID := int64(123)

	sub, err := NewSubscription(email, repoID)
	require.NoError(t, err)

	assert.Equal(t, email, sub.Email)
	assert.Equal(t, repoID, sub.RepositoryID)
	assert.NotEmpty(t, sub.ConfirmToken)
	assert.NotEmpty(t, sub.UnsubscribeToken)
	assert.False(t, sub.Confirmed)
}
