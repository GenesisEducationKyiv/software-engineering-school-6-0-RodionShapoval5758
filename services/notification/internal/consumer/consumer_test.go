package consumer

import (
	"encoding/json"
	"errors"
	"testing"

	"GithubReleaseNotificationAPI/contract"
)

type stubMailer struct {
	confirmCalled bool
	confirmArgs   [3]string
	releaseCalled bool
	releaseArgs   [5]string
	err           error
}

func (s *stubMailer) SendConfirmation(toEmail, repoName, confirmToken string) error {
	s.confirmCalled = true
	s.confirmArgs = [3]string{toEmail, repoName, confirmToken}
	return s.err
}

func (s *stubMailer) SendRelease(toEmail, unsubscribeToken, releaseTag, releaseName, releaseURL string) error {
	s.releaseCalled = true
	s.releaseArgs = [5]string{toEmail, unsubscribeToken, releaseTag, releaseName, releaseURL}
	return s.err
}

func TestProcessMessage_Confirmation(t *testing.T) {
	ev := contract.ConfirmationRequested{
		Email:        "user@example.com",
		RepoName:     "owner/repo",
		ConfirmToken: "tok123",
	}
	data, _ := json.Marshal(ev)

	m := &stubMailer{}
	ack, term := processMessage(contract.SubjectConfirmation, data, m)

	if !ack || term {
		t.Fatalf("expected ack=true term=false, got ack=%v term=%v", ack, term)
	}
	if !m.confirmCalled {
		t.Fatal("SendConfirmation not called")
	}
	if m.confirmArgs != [3]string{"user@example.com", "owner/repo", "tok123"} {
		t.Fatalf("unexpected args: %v", m.confirmArgs)
	}
}

func TestProcessMessage_Release(t *testing.T) {
	ev := contract.ReleasePublished{
		Email:            "user@example.com",
		UnsubscribeToken: "unsub456",
		ReleaseTag:       "v1.2.3",
		ReleaseName:      "My Release",
		ReleaseURL:       "https://github.com/owner/repo/releases/tag/v1.2.3",
	}
	data, _ := json.Marshal(ev)

	m := &stubMailer{}
	ack, term := processMessage(contract.SubjectRelease, data, m)

	if !ack || term {
		t.Fatalf("expected ack=true term=false, got ack=%v term=%v", ack, term)
	}
	if !m.releaseCalled {
		t.Fatal("SendRelease not called")
	}
	want := [5]string{"user@example.com", "unsub456", "v1.2.3", "My Release", "https://github.com/owner/repo/releases/tag/v1.2.3"}
	if m.releaseArgs != want {
		t.Fatalf("unexpected args: %v", m.releaseArgs)
	}
}

func TestProcessMessage_SendConfirmationFailure(t *testing.T) {
	ev := contract.ConfirmationRequested{Email: "a@b.com", RepoName: "r", ConfirmToken: "t"}
	data, _ := json.Marshal(ev)

	m := &stubMailer{err: errors.New("smtp down")}
	ack, term := processMessage(contract.SubjectConfirmation, data, m)

	if ack || term {
		t.Fatalf("expected ack=false term=false on send failure, got ack=%v term=%v", ack, term)
	}
}

func TestProcessMessage_SendReleaseFailure(t *testing.T) {
	ev := contract.ReleasePublished{Email: "a@b.com", UnsubscribeToken: "u", ReleaseTag: "v1", ReleaseName: "R", ReleaseURL: "http://x"}
	data, _ := json.Marshal(ev)

	m := &stubMailer{err: errors.New("smtp down")}
	ack, term := processMessage(contract.SubjectRelease, data, m)

	if ack || term {
		t.Fatalf("expected ack=false term=false on send failure, got ack=%v term=%v", ack, term)
	}
}

func TestProcessMessage_BadJSONConfirmation(t *testing.T) {
	m := &stubMailer{}
	ack, term := processMessage(contract.SubjectConfirmation, []byte("not-json"), m)

	if ack || !term {
		t.Fatalf("expected ack=false term=true on bad JSON, got ack=%v term=%v", ack, term)
	}
	if m.confirmCalled {
		t.Fatal("SendConfirmation should not have been called")
	}
}

func TestProcessMessage_BadJSONRelease(t *testing.T) {
	m := &stubMailer{}
	ack, term := processMessage(contract.SubjectRelease, []byte("{bad"), m)

	if ack || !term {
		t.Fatalf("expected ack=false term=true on bad JSON, got ack=%v term=%v", ack, term)
	}
}

func TestProcessMessage_UnknownSubject(t *testing.T) {
	m := &stubMailer{}
	ack, term := processMessage("notifications.something-else", []byte("{}"), m)

	if !ack || term {
		t.Fatalf("expected ack=true term=false for unknown subject, got ack=%v term=%v", ack, term)
	}
	if m.confirmCalled || m.releaseCalled {
		t.Fatal("no send should be called for unknown subject")
	}
}
