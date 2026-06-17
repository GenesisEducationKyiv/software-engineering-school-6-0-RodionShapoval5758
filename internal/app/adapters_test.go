package app

import (
	"testing"

	"GithubReleaseNotificationAPI/internal/fanout"
	"GithubReleaseNotificationAPI/internal/subscription"
)

func TestSubsToRecipients_MultipleSubscribers(t *testing.T) {
	subs := []subscription.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "tok-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "tok-bob"},
	}

	got := subsToRecipients(subs)

	if len(got) != len(subs) {
		t.Fatalf("want %d recipients, got %d", len(subs), len(got))
	}
	for i, want := range []fanout.Recipient{
		{Email: "alice@example.com", UnsubscribeToken: "tok-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "tok-bob"},
	} {
		if got[i] != want {
			t.Errorf("recipient %d: want %+v, got %+v", i, want, got[i])
		}
	}
}

func TestSubsToRecipients_EmptySlice(t *testing.T) {
	got := subsToRecipients(nil)
	if len(got) != 0 {
		t.Fatalf("want 0 recipients, got %d", len(got))
	}
}
