package mailer

import (
	"os"
	"strings"
	"testing"
)

func TestSendConfirmation_NoAuth(t *testing.T) {
	if os.Getenv("SMTP_HOST") == "" {
		t.Skip("SMTP_HOST not set, skipping live SMTP test")
	}

	m := NewMailer(
		os.Getenv("SMTP_HOST"),
		"1025",
		"", "",
		"noreply@localhost",
		"http://localhost:8080",
	)

	err := m.SendConfirmation("test@example.com", "owner/repo", "token123")
	if err != nil {
		t.Fatalf("SendConfirmation failed: %v", err)
	}
}

func TestSendRelease_NoAuth(t *testing.T) {
	if os.Getenv("SMTP_HOST") == "" {
		t.Skip("SMTP_HOST not set, skipping live SMTP test")
	}

	m := NewMailer(
		os.Getenv("SMTP_HOST"),
		"1025",
		"", "",
		"noreply@localhost",
		"http://localhost:8080",
	)

	err := m.SendRelease("test@example.com", "unsub-token-abc", "v2.0.0", "Awesome Release", "https://github.com/owner/repo/releases/tag/v2.0.0")
	if err != nil {
		t.Fatalf("SendRelease failed: %v", err)
	}
}

func TestRenderConfirmationEmail(t *testing.T) {
	repoName := "owner/repo"
	confirmLink := "http://localhost/confirm/abc123"

	out, err := renderConfirmationEmail(repoName, confirmLink)
	if err != nil {
		t.Fatalf("renderConfirmationEmail returned error: %v", err)
	}

	if !strings.Contains(out, repoName) {
		t.Errorf("output missing repo name %q", repoName)
	}
	if !strings.Contains(out, confirmLink) {
		t.Errorf("output missing confirm link %q", confirmLink)
	}
}

func TestRenderReleaseEmail(t *testing.T) {
	releaseName := "My Release"
	tag := "v1.2.3"
	releaseURL := "https://github.com/x"
	unsubscribeLink := "http://localhost/unsubscribe/tok"

	out, err := renderReleaseEmail(releaseName, tag, releaseURL, unsubscribeLink)
	if err != nil {
		t.Fatalf("renderReleaseEmail returned error: %v", err)
	}

	for _, want := range []string{tag, releaseName, releaseURL, unsubscribeLink} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}
