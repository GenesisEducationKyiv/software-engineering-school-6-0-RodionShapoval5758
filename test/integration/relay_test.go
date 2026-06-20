//go:build integration

package integration_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/internal/outbox"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
)

type RelaySuite struct {
	suite.Suite
	stream jetstream.Stream
}

func TestRelaySuite(t *testing.T) {
	suite.Run(t, new(RelaySuite))
}

func (s *RelaySuite) SetupSuite() {
	stream, err := testJS.Stream(context.Background(), contract.StreamName)
	s.Require().NoError(err)
	s.stream = stream
}

func (s *RelaySuite) SetupTest() {
	ctx := context.Background()
	_, err := testPool.Exec(ctx, "TRUNCATE outbox")
	s.Require().NoError(err)
	s.Require().NoError(s.stream.Purge(ctx))
}

func (s *RelaySuite) runRelay() (cancel context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	relay := outbox.NewRelay(testPool, testJS, outbox.NewStore())
	go relay.Run(ctx)
	return cancel
}

func (s *RelaySuite) waitForMsgs(n uint64) {
	s.T().Helper()
	s.Require().Eventually(func() bool {
		info, err := s.stream.Info(context.Background())
		return err == nil && info.State.Msgs == n
	}, 5*time.Second, 50*time.Millisecond)
}

func (s *RelaySuite) TestPublishesPendingRow() {
	payload := []byte(`{"email":"a@b.com","repo_name":"owner/repo","confirm_token":"tok"}`)

	store := outbox.NewStore()
	s.Require().NoError(store.Insert(context.Background(), testPool, contract.SubjectConfirmation, payload))

	cancel := s.runRelay()
	defer cancel()

	s.waitForMsgs(1)

	msg, err := s.stream.GetLastMsgForSubject(context.Background(), contract.SubjectConfirmation)
	s.Require().NoError(err)
	s.Equal(contract.SubjectConfirmation, msg.Subject)
	s.Equal(payload, msg.Data)
}

func (s *RelaySuite) TestMarksPublishedAt() {
	store := outbox.NewStore()
	s.Require().NoError(store.Insert(context.Background(), testPool, contract.SubjectConfirmation, []byte(`{}`)))

	cancel := s.runRelay()
	defer cancel()

	s.waitForMsgs(1)

	var publishedAt *time.Time
	s.Require().Eventually(func() bool {
		row := testPool.QueryRow(context.Background(), "SELECT published_at FROM outbox LIMIT 1")
		return row.Scan(&publishedAt) == nil && publishedAt != nil
	}, 3*time.Second, 50*time.Millisecond)
}

func (s *RelaySuite) TestDedupByMsgID() {
	store := outbox.NewStore()
	s.Require().NoError(store.Insert(context.Background(), testPool, contract.SubjectConfirmation, []byte(`{}`)))

	var outboxID int64
	s.Require().NoError(testPool.QueryRow(context.Background(), "SELECT id FROM outbox LIMIT 1").Scan(&outboxID))

	cancel := s.runRelay()
	defer cancel()

	s.waitForMsgs(1)
	cancel()

	_, err := testJS.Publish(
		context.Background(),
		contract.SubjectConfirmation,
		[]byte(`{}`),
		jetstream.WithMsgID(strconv.FormatInt(outboxID, 10)),
	)
	s.Require().NoError(err)

	info, err := s.stream.Info(context.Background())
	s.Require().NoError(err)
	s.Equal(uint64(1), info.State.Msgs, "duplicate MsgID must not add a second message")
}

func (s *RelaySuite) TestOnlyPendingRows() {
	store := outbox.NewStore()
	s.Require().NoError(store.Insert(context.Background(), testPool, contract.SubjectConfirmation, []byte(`{}`)))
	_, err := testPool.Exec(context.Background(), "UPDATE outbox SET published_at = now()")
	s.Require().NoError(err)

	s.Require().NoError(store.Insert(context.Background(), testPool, contract.SubjectRelease, []byte(`{}`)))

	cancel := s.runRelay()
	defer cancel()

	s.waitForMsgs(1)

	msg, err := s.stream.GetLastMsgForSubject(context.Background(), contract.SubjectRelease)
	s.Require().NoError(err)
	s.Equal(contract.SubjectRelease, msg.Subject)
}
