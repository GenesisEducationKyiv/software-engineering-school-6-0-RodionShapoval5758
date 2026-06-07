package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type call struct {
	subject string
	data    []byte
}

type stubPublisher struct {
	calls   []call
	failOn  map[int]error
	callIdx int
}

func (s *stubPublisher) Publish(_ context.Context, subject string, data []byte, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	idx := s.callIdx
	s.callIdx++
	s.calls = append(s.calls, call{subject: subject, data: data})

	if err, ok := s.failOn[idx]; ok {
		return nil, err
	}

	return &jetstream.PubAck{}, nil
}

func TestSendConfirmation_PublishesCorrectPayload(t *testing.T) {
	stub := &stubPublisher{}
	p := &Publisher{pub: stub}

	err := p.SendConfirmation("user@example.com", "owner/repo", "tok123")
	require.NoError(t, err)

	require.Len(t, stub.calls, 1)
	assert.Equal(t, contract.SubjectConfirmation, stub.calls[0].subject)

	var got contract.ConfirmationRequested
	require.NoError(t, json.Unmarshal(stub.calls[0].data, &got))
	assert.Equal(t, "user@example.com", got.Email)
	assert.Equal(t, "owner/repo", got.RepoName)
	assert.Equal(t, "tok123", got.ConfirmToken)
}

func TestSendReleaseEmails_TwoRecipients(t *testing.T) {
	stub := &stubPublisher{}
	p := &Publisher{pub: stub}

	recipients := []ReleaseRecipient{
		{Email: "a@example.com", UnsubscribeToken: "unsub-a"},
		{Email: "b@example.com", UnsubscribeToken: "unsub-b"},
	}
	release := ReleaseInfo{Tag: "v1.0.0", Name: "Repo", URL: "https://example.com/releases/v1.0.0"}

	err := p.SendReleaseEmails(recipients, release)
	require.NoError(t, err)

	require.Len(t, stub.calls, 2)

	for i, c := range stub.calls {
		assert.Equal(t, contract.SubjectRelease, c.subject)

		var got contract.ReleasePublished
		require.NoError(t, json.Unmarshal(c.data, &got))
		assert.Equal(t, recipients[i].Email, got.Email)
		assert.Equal(t, recipients[i].UnsubscribeToken, got.UnsubscribeToken)
		assert.Equal(t, release.Tag, got.ReleaseTag)
		assert.Equal(t, release.Name, got.ReleaseName)
		assert.Equal(t, release.URL, got.ReleaseURL)
	}
}

func TestSendReleaseEmails_ZeroRecipients(t *testing.T) {
	stub := &stubPublisher{}
	p := &Publisher{pub: stub}

	err := p.SendReleaseEmails(nil, ReleaseInfo{})
	require.NoError(t, err)
	assert.Empty(t, stub.calls)
}

func TestSendReleaseEmails_OneFailStillAttemptsRest(t *testing.T) {
	publishErr := errors.New("broker down")
	stub := &stubPublisher{
		failOn: map[int]error{0: publishErr},
	}
	p := &Publisher{pub: stub}

	recipients := []ReleaseRecipient{
		{Email: "fail@example.com", UnsubscribeToken: "tok-fail"},
		{Email: "ok@example.com", UnsubscribeToken: "tok-ok"},
	}

	err := p.SendReleaseEmails(recipients, ReleaseInfo{Tag: "v1.0.0", Name: "R", URL: "u"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail@example.com")

	assert.Len(t, stub.calls, 2, "must attempt second recipient even after first fails")
}
