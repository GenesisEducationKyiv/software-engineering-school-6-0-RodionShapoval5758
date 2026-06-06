package watcher_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/notification"

	"github.com/stretchr/testify/mock"
)

func (s *WatcherTestSuite) TestReleaseNotifier_ListError() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(nil, errors.New("db error"))

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.Error(err)
	s.assertExpectations()
}

func (s *WatcherTestSuite) TestReleaseNotifier_NoSubscribers() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return([]domain.Subscription{}, nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}

func (s *WatcherTestSuite) TestReleaseNotifier_MultipleSubscribers() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0", Name: "Release 1", URL: "https://github.com"}
	subs := []domain.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}
	expectedRecipients := []notification.ReleaseRecipient{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}
	expectedRelease := notification.ReleaseInfo{Tag: "v1.0.0", Name: "Release 1", URL: "https://github.com"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(subs, nil)
	s.smtp.On("SendReleaseEmails", expectedRecipients, expectedRelease).Return(nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}

func (s *WatcherTestSuite) TestReleaseNotifier_SMTPError_Propagates() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0", Name: "Release 1", URL: "https://github.com"}
	subs := []domain.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}
	expectedRecipients := []notification.ReleaseRecipient{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}
	expectedRelease := notification.ReleaseInfo{Tag: "v1.0.0", Name: "Release 1", URL: "https://github.com"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(subs, nil)
	s.smtp.On("SendReleaseEmails", expectedRecipients, expectedRelease).
		Return(errors.New("partial smtp failure"))

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.Error(err)
	s.assertExpectations()
}
