package contract_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"GithubReleaseNotificationAPI/contract"
)

func TestConfirmationRequestedJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		event contract.ConfirmationRequested
	}{
		{
			name: "standard confirmation event",
			event: contract.ConfirmationRequested{
				Email:        "user@example.com",
				RepoName:     "golang/go",
				ConfirmToken: "abc123def456",
			},
		},
		{
			name: "confirmation with special characters in email",
			event: contract.ConfirmationRequested{
				Email:        "user+tag@sub.example.co.uk",
				RepoName:     "owner/repo-name",
				ConfirmToken: "token_with_underscores",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			require.NoError(t, err)

			var unmarshaled contract.ConfirmationRequested
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.event, unmarshaled)
		})
	}
}

func TestReleasePublishedJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		event contract.ReleasePublished
	}{
		{
			name: "standard release event",
			event: contract.ReleasePublished{
				Email:            "user@example.com",
				UnsubscribeToken: "unsub123def456",
				ReleaseTag:       "v1.2.3",
				ReleaseName:      "Version 1.2.3 - Stable Release",
				ReleaseURL:       "https://github.com/owner/repo/releases/tag/v1.2.3",
			},
		},
		{
			name: "release with special characters in name",
			event: contract.ReleasePublished{
				Email:            "dev+notify@company.com",
				UnsubscribeToken: "token-with-dashes",
				ReleaseTag:       "v2.0.0-beta.1",
				ReleaseName:      "Release 2.0.0 Beta (March 2025)",
				ReleaseURL:       "https://github.com/org-name/my-repo/releases/tag/v2.0.0-beta.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			require.NoError(t, err)

			var unmarshaled contract.ReleasePublished
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.event, unmarshaled)
		})
	}
}
