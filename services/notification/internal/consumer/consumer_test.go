package consumer

import (
	"errors"
	"testing"

	"GithubReleaseNotificationAPI/contract"

	"github.com/stretchr/testify/assert"
)

type mockMailer struct {
	confirmErr error
	releaseErr error
}

func (m *mockMailer) SendConfirmation(toEmail, repoName, confirmToken string) error {
	return m.confirmErr
}

func (m *mockMailer) SendRelease(toEmail, unsubscribeToken, releaseTag, releaseName, releaseURL string) error {
	return m.releaseErr
}

var (
	validConfirmation = []byte(`{"email":"a@b.com","repo_name":"owner/repo","confirm_token":"tok"}`)
	validRelease      = []byte(`{"email":"a@b.com","unsubscribe_token":"tok","release_tag":"v1","release_name":"Release v1","release_url":"https://example.com"}`)
	badJSON           = []byte(`not json`)
)

func TestDecideAction(t *testing.T) {
	tests := []struct {
		name     string
		oc       outcome
		num      uint64
		expected action
	}{
		{"ack always acks", outcomeAck, 1, actionAck},
		{"poison always dlqs", outcomePoison, 1, actionDLQ},
		{"retry below max naks", outcomeRetry, maxDeliver - 1, actionNak},
		{"retry at max dlqs", outcomeRetry, maxDeliver, actionDLQ},
		{"retry above max dlqs", outcomeRetry, maxDeliver + 1, actionDLQ},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, decideAction(tt.oc, tt.num))
		})
	}
}

func TestProcessMessage(t *testing.T) {
	smtpErr := errors.New("smtp down")

	tests := []struct {
		name            string
		subject         string
		data            []byte
		confirmErr      error
		releaseErr      error
		expectedOutcome outcome
	}{
		{"confirmation success", contract.SubjectConfirmation, validConfirmation, nil, nil, outcomeAck},
		{"confirmation bad json", contract.SubjectConfirmation, badJSON, nil, nil, outcomePoison},
		{"confirmation mailer error", contract.SubjectConfirmation, validConfirmation, smtpErr, nil, outcomeRetry},
		{"release success", contract.SubjectRelease, validRelease, nil, nil, outcomeAck},
		{"release bad json", contract.SubjectRelease, badJSON, nil, nil, outcomePoison},
		{"release mailer error", contract.SubjectRelease, validRelease, nil, smtpErr, outcomeRetry},
		{"unknown subject acks", "notifications.unknown", validRelease, nil, nil, outcomeAck},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockMailer{confirmErr: tt.confirmErr, releaseErr: tt.releaseErr}
			oc, _ := processMessage(tt.subject, tt.data, m)
			assert.Equal(t, tt.expectedOutcome, oc)
		})
	}
}

func TestBuildDeadLetter(t *testing.T) {
	data := []byte(`{"email":"a@b.com"}`)
	dl := buildDeadLetter(contract.SubjectRelease, data, "smtp down", 5)

	assert.Equal(t, contract.SubjectRelease, dl.OriginalSubject)
	assert.Equal(t, "smtp down", dl.Reason)
	assert.Equal(t, uint64(5), dl.Attempts)
	assert.NotZero(t, dl.FailedAt)
	assert.JSONEq(t, `{"email":"a@b.com"}`, string(dl.Payload))
}
