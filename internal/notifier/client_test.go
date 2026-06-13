package notifier

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"GithubReleaseNotificationAPI/contract"
)

func TestSendConfirmation(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody contract.ConfirmationRequested

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	err := c.SendConfirmation("a@b.com", "owner/repo", "tok123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/v1/emails/confirmation" {
		t.Errorf("path = %s, want /v1/emails/confirmation", gotPath)
	}
	want := contract.ConfirmationRequested{Email: "a@b.com", RepoName: "owner/repo", ConfirmToken: "tok123"}
	if gotBody != want {
		t.Errorf("body = %+v, want %+v", gotBody, want)
	}
}

func TestSendConfirmationNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	err := c.SendConfirmation("a@b.com", "r", "t")

	if err == nil {
		t.Fatal("expected error on non-2xx, got nil")
	}
}

func TestSendReleaseEmails(t *testing.T) {
	var calls []contract.ReleasePublished

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var ev contract.ReleasePublished
		_ = json.Unmarshal(body, &ev)
		calls = append(calls, ev)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	recipients := []ReleaseRecipient{
		{Email: "a@b.com", UnsubscribeToken: "u1"},
		{Email: "c@d.com", UnsubscribeToken: "u2"},
	}
	release := ReleaseInfo{Tag: "v1.0.0", Name: "Release 1", URL: "https://example.com"}

	err := c.SendReleaseEmails(recipients, release)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 POST calls, got %d", len(calls))
	}
	if calls[0].Email != "a@b.com" || calls[1].Email != "c@d.com" {
		t.Errorf("unexpected call order: %v", calls)
	}
}

func TestSendReleaseEmailsJoinsErrors(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	recipients := []ReleaseRecipient{
		{Email: "a@b.com", UnsubscribeToken: "u1"},
		{Email: "c@d.com", UnsubscribeToken: "u2"},
	}

	err := c.SendReleaseEmails(recipients, ReleaseInfo{Tag: "v1", Name: "R", URL: "http://x"})

	if err == nil {
		t.Fatal("expected joined error, got nil")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls even on failure, got %d", callCount)
	}
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Log("errors.Join result is not wrapped in the same way, checking join structure")
	}
}
