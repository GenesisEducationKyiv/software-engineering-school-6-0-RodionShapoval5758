package notifier_test

import (
	"testing"

	"GithubReleaseNotificationAPI/internal/notifier"

	"github.com/stretchr/testify/suite"
)

type NotifierTestSuite struct {
	suite.Suite

	smtp    *mockSmtpClient
	subRepo *mockSubscriptionRepository

	releaseNotifier *notifier.ReleaseNotifier
}

func (s *NotifierTestSuite) SetupTest() {
	s.smtp = new(mockSmtpClient)
	s.subRepo = new(mockSubscriptionRepository)

	s.releaseNotifier = notifier.NewReleaseNotifier(s.smtp, s.subRepo)
}

func (s *NotifierTestSuite) assertExpectations() {
	s.smtp.AssertExpectations(s.T())
	s.subRepo.AssertExpectations(s.T())
}

func TestNotifierTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}
