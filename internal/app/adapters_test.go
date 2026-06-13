package app_test

import (
	"context"
	"errors"
	"testing"

	"GithubReleaseNotificationAPI/internal/app"
	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/subscription"
)

type stubSubProvider struct {
	subs []subscription.Subscription
	err  error
}

func (s *stubSubProvider) ListConfirmedByRepositoryID(_ context.Context, _ int64) ([]subscription.Subscription, error) {
	return s.subs, s.err
}

func TestConfirmedSubReader_mapsTwoFields(t *testing.T) {
	stub := &stubSubProvider{subs: []subscription.Subscription{
		{Email: "a@b.com", UnsubscribeToken: "tok1", ConfirmToken: "secret1"},
		{Email: "c@d.com", UnsubscribeToken: "tok2", ConfirmToken: "secret2"},
	}}

	got, err := app.NewConfirmedSubReader(stub).ListConfirmedByRepositoryID(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}

	want := []monitoring.ConfirmedSubscriber{
		{Email: "a@b.com", UnsubscribeToken: "tok1"},
		{Email: "c@d.com", UnsubscribeToken: "tok2"},
	}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] got %+v want %+v", i, got[i], w)
		}
	}
}

func TestConfirmedSubReader_emptySlice(t *testing.T) {
	stub := &stubSubProvider{subs: []subscription.Subscription{}}

	got, err := app.NewConfirmedSubReader(stub).ListConfirmedByRepositoryID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestConfirmedSubReader_propagatesError(t *testing.T) {
	sentinel := errors.New("db down")
	stub := &stubSubProvider{err: sentinel}

	_, err := app.NewConfirmedSubReader(stub).ListConfirmedByRepositoryID(context.Background(), 1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want %v", err, sentinel)
	}
}
