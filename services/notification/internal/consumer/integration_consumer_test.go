//go:build integration

package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
)

type sigMailer struct {
	err  error
	once sync.Once
	ch   chan struct{}
}

func newSigMailer(err error) *sigMailer {
	return &sigMailer{err: err, ch: make(chan struct{})}
}

func (m *sigMailer) SendConfirmation(_, _, _ string) error {
	m.once.Do(func() { close(m.ch) })
	return m.err
}

func (m *sigMailer) SendRelease(_, _, _, _, _ string) error {
	m.once.Do(func() { close(m.ch) })
	return m.err
}

type ConsumerIntegrationSuite struct {
	suite.Suite
	mainStream jetstream.Stream
	dlqStream  jetstream.Stream
}

func TestConsumerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ConsumerIntegrationSuite))
}

func (s *ConsumerIntegrationSuite) SetupSuite() {
	ctx := context.Background()

	stream, err := testJS.Stream(ctx, contract.StreamName)
	s.Require().NoError(err)
	s.mainStream = stream

	dlq, err := testJS.Stream(ctx, contract.StreamDLQ)
	s.Require().NoError(err)
	s.dlqStream = dlq
}

func (s *ConsumerIntegrationSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.mainStream.Purge(ctx))
	s.Require().NoError(s.dlqStream.Purge(ctx))
	_ = testJS.DeleteConsumer(ctx, contract.StreamName, "notification-consumer")
}

func (s *ConsumerIntegrationSuite) startConsumer(m Mailer) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	c := New(testJS, m)
	go func() { _ = c.Start(ctx) }()
	return cancel
}

func (s *ConsumerIntegrationSuite) waitMainDrain() {
	s.T().Helper()
	s.Require().Eventually(func() bool {
		info, err := s.mainStream.Info(context.Background())
		return err == nil && info.State.Msgs == 0
	}, 10*time.Second, 50*time.Millisecond)
}

func (s *ConsumerIntegrationSuite) waitDLQMsg() {
	s.T().Helper()
	s.Require().Eventually(func() bool {
		info, err := s.dlqStream.Info(context.Background())
		return err == nil && info.State.Msgs == 1
	}, 10*time.Second, 50*time.Millisecond)
}

func (s *ConsumerIntegrationSuite) TestValidConfirmation_Acks() {
	mailer := newSigMailer(nil)
	cancel := s.startConsumer(mailer)
	defer cancel()

	_, err := testJS.Publish(context.Background(), contract.SubjectConfirmation, validConfirmation)
	s.Require().NoError(err)

	select {
	case <-mailer.ch:
	case <-time.After(5 * time.Second):
		s.Fail("mailer SendConfirmation not called")
	}

	s.waitMainDrain()

	info, err := s.dlqStream.Info(context.Background())
	s.Require().NoError(err)
	s.Equal(uint64(0), info.State.Msgs)
}

func (s *ConsumerIntegrationSuite) TestPoisonPayload_ImmediateDLQ() {
	mailer := newSigMailer(nil)
	cancel := s.startConsumer(mailer)
	defer cancel()

	_, err := testJS.Publish(context.Background(), contract.SubjectConfirmation, badJSON)
	s.Require().NoError(err)

	s.waitDLQMsg()
	s.waitMainDrain()

	select {
	case <-mailer.ch:
		s.Fail("mailer must not be called for poison payload")
	default:
	}

	raw, err := s.dlqStream.GetLastMsgForSubject(context.Background(), contract.SubjectDead)
	s.Require().NoError(err)

	var dl contract.DeadLetter
	s.Require().NoError(json.Unmarshal(raw.Data, &dl))
	s.Equal(contract.SubjectConfirmation, dl.OriginalSubject)
	s.Equal("unmarshalable", dl.Reason)
	s.Equal(uint64(1), dl.Attempts)
	s.Contains(string(dl.Payload), "not json") // stored as JSON string since payload is not valid JSON
	s.False(dl.FailedAt.IsZero())
}

func (s *ConsumerIntegrationSuite) TestMailerError_RetriesThenDLQ() {
	smtpErr := errors.New("smtp down")
	mailer := newSigMailer(smtpErr)
	cancel := s.startConsumer(mailer)
	defer cancel()

	_, err := testJS.Publish(context.Background(), contract.SubjectConfirmation, validConfirmation)
	s.Require().NoError(err)

	// NAK redelivery is fast (no explicit delay in consumer); budget 60s for 5 retries
	s.Require().Eventually(func() bool {
		info, err := s.dlqStream.Info(context.Background())
		return err == nil && info.State.Msgs == 1
	}, 60*time.Second, 200*time.Millisecond, "message should exhaust retries and land in DLQ")

	s.waitMainDrain()

	raw, err := s.dlqStream.GetLastMsgForSubject(context.Background(), contract.SubjectDead)
	s.Require().NoError(err)

	var dl contract.DeadLetter
	s.Require().NoError(json.Unmarshal(raw.Data, &dl))
	s.Equal(contract.SubjectConfirmation, dl.OriginalSubject)
	s.Equal(smtpErr.Error(), dl.Reason)
	s.Equal(uint64(maxDeliver), dl.Attempts)
}
