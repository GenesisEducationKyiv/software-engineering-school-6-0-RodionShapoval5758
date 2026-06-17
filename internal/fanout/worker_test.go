package fanout

import (
	"encoding/json"
	"testing"

	"GithubReleaseNotificationAPI/contract"
)

func TestBuildEvents_MultipleRecipients(t *testing.T) {
	tests := []struct {
		name     string
		release  DetectedRelease
		recs     []Recipient
		wantLen  int
		wantTags []string
	}{
		{
			name: "two recipients get two payloads with correct fields",
			release: DetectedRelease{
				ReleaseTag:  "v1.0.0",
				ReleaseName: "First Release",
				ReleaseURL:  "https://github.com/owner/repo/releases/tag/v1.0.0",
			},
			recs: []Recipient{
				{Email: "alice@example.com", UnsubscribeToken: "tok-alice"},
				{Email: "bob@example.com", UnsubscribeToken: "tok-bob"},
			},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payloads, err := buildEvents(tc.release, tc.recs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(payloads) != tc.wantLen {
				t.Fatalf("want %d payloads, got %d", tc.wantLen, len(payloads))
			}

			for i, raw := range payloads {
				var ev contract.ReleaseDetected
				if err := json.Unmarshal(raw, &ev); err != nil {
					t.Fatalf("payload %d: unmarshal error: %v", i, err)
				}
				if ev.Email != tc.recs[i].Email {
					t.Errorf("payload %d: email: want %q, got %q", i, tc.recs[i].Email, ev.Email)
				}
				if ev.UnsubscribeToken != tc.recs[i].UnsubscribeToken {
					t.Errorf("payload %d: unsubscribe_token: want %q, got %q", i, tc.recs[i].UnsubscribeToken, ev.UnsubscribeToken)
				}
				if ev.ReleaseTag != tc.release.ReleaseTag {
					t.Errorf("payload %d: release_tag: want %q, got %q", i, tc.release.ReleaseTag, ev.ReleaseTag)
				}
				if ev.ReleaseName != tc.release.ReleaseName {
					t.Errorf("payload %d: release_name: want %q, got %q", i, tc.release.ReleaseName, ev.ReleaseName)
				}
				if ev.ReleaseURL != tc.release.ReleaseURL {
					t.Errorf("payload %d: release_url: want %q, got %q", i, tc.release.ReleaseURL, ev.ReleaseURL)
				}
			}
		})
	}
}

func TestBuildEvents_ZeroRecipients(t *testing.T) {
	release := DetectedRelease{
		ReleaseTag:  "v2.0.0",
		ReleaseName: "Second Release",
		ReleaseURL:  "https://github.com/owner/repo/releases/tag/v2.0.0",
	}

	payloads, err := buildEvents(release, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payloads) != 0 {
		t.Fatalf("want 0 payloads, got %d", len(payloads))
	}
}
