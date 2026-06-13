package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestHandleConfirmation(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		mailerErr  error
		wantStatus int
		wantCalled bool
		wantArgs   [3]string
	}{
		{
			name: "success",
			body: contract.ConfirmationRequested{
				Email: "a@b.com", RepoName: "owner/repo", ConfirmToken: "tok",
			},
			wantStatus: http.StatusAccepted,
			wantCalled: true,
			wantArgs:   [3]string{"a@b.com", "owner/repo", "tok"},
		},
		{
			name:       "bad json",
			body:       "not-json",
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:       "mailer error returns 502",
			body:       contract.ConfirmationRequested{Email: "a@b.com", RepoName: "r", ConfirmToken: "t"},
			mailerErr:  errors.New("smtp down"),
			wantStatus: http.StatusBadGateway,
			wantCalled: true,
			wantArgs:   [3]string{"a@b.com", "r", "t"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &stubMailer{err: tc.mailerErr}
			h := New(m)

			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(tc.body)

			req := httptest.NewRequest(http.MethodPost, "/v1/emails/confirmation", &buf)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if m.confirmCalled != tc.wantCalled {
				t.Errorf("confirmCalled = %v, want %v", m.confirmCalled, tc.wantCalled)
			}
			if tc.wantCalled && m.confirmArgs != tc.wantArgs {
				t.Errorf("args = %v, want %v", m.confirmArgs, tc.wantArgs)
			}
		})
	}
}

func TestHandleRelease(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		mailerErr  error
		wantStatus int
		wantCalled bool
		wantArgs   [5]string
	}{
		{
			name: "success",
			body: contract.ReleasePublished{
				Email: "a@b.com", UnsubscribeToken: "unsub", ReleaseTag: "v1.0.0",
				ReleaseName: "Release 1", ReleaseURL: "https://example.com",
			},
			wantStatus: http.StatusAccepted,
			wantCalled: true,
			wantArgs:   [5]string{"a@b.com", "unsub", "v1.0.0", "Release 1", "https://example.com"},
		},
		{
			name:       "bad json",
			body:       "not-json",
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name: "mailer error returns 502",
			body: contract.ReleasePublished{
				Email: "a@b.com", UnsubscribeToken: "u", ReleaseTag: "v1",
				ReleaseName: "R", ReleaseURL: "http://x",
			},
			mailerErr:  errors.New("smtp down"),
			wantStatus: http.StatusBadGateway,
			wantCalled: true,
			wantArgs:   [5]string{"a@b.com", "u", "v1", "R", "http://x"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &stubMailer{err: tc.mailerErr}
			h := New(m)

			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(tc.body)

			req := httptest.NewRequest(http.MethodPost, "/v1/emails/release", &buf)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if m.releaseCalled != tc.wantCalled {
				t.Errorf("releaseCalled = %v, want %v", m.releaseCalled, tc.wantCalled)
			}
			if tc.wantCalled && m.releaseArgs != tc.wantArgs {
				t.Errorf("args = %v, want %v", m.releaseArgs, tc.wantArgs)
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	h := New(&stubMailer{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health status = %d, want 200", rec.Code)
	}
}
