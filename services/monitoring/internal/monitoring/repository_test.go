package monitoring

import "testing"

func TestTrackedRepoHasNewRelease(t *testing.T) {
	tests := []struct {
		name         string
		lastSeenTag  string
		incomingTag  string
		expectNewRel bool
	}{
		{
			name:         "no cursor yet (empty LastSeenTag)",
			lastSeenTag:  "",
			incomingTag:  "v1.0.0",
			expectNewRel: true,
		},
		{
			name:         "same tag, no release",
			lastSeenTag:  "v1.0.0",
			incomingTag:  "v1.0.0",
			expectNewRel: false,
		},
		{
			name:         "new release",
			lastSeenTag:  "v1.0.0",
			incomingTag:  "v2.0.0",
			expectNewRel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := TrackedRepo{
				ID:          1,
				FullName:    "test/repo",
				LastSeenTag: tt.lastSeenTag,
			}
			got := repo.HasNewRelease(tt.incomingTag)
			if got != tt.expectNewRel {
				t.Errorf("HasNewRelease(%q) = %v, want %v", tt.incomingTag, got, tt.expectNewRel)
			}
		})
	}
}
