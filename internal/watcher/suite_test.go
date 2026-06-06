package watcher_test

import (
	"testing"

	"GithubReleaseNotificationAPI/internal/watcher"

	"github.com/stretchr/testify/suite"
)

type WatcherTestSuite struct {
	suite.Suite

	smtp    *mockSmtpClient
	subRepo *mockSubscriptionRepository

	releaseNotifier *watcher.ReleaseNotifier
}

func (s *WatcherTestSuite) SetupTest() {
	s.smtp = new(mockSmtpClient)
	s.subRepo = new(mockSubscriptionRepository)

	s.releaseNotifier = watcher.NewReleaseNotifier(s.smtp, s.subRepo)
}

func (s *WatcherTestSuite) assertExpectations() {
	s.smtp.AssertExpectations(s.T())
	s.subRepo.AssertExpectations(s.T())
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
