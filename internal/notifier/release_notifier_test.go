package notifier_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/domain"

	"github.com/stretchr/testify/mock"
)

func (s *NotifierTestSuite) TestReleaseNotifier_ListError() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(nil, errors.New("db error"))

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.Error(err)
	s.assertExpectations()
}

func (s *NotifierTestSuite) TestReleaseNotifier_NoSubscribers() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return([]domain.Subscription{}, nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}

func (s *NotifierTestSuite) TestReleaseNotifier_SingleSubscriber() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}
	sub := domain.Subscription{Email: "user@example.com", UnsubscribeToken: "unsub-token"}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return([]domain.Subscription{sub}, nil)
	s.smtp.On("SendReleaseNotification", "user@example.com", "unsub-token", release).
		Return(nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}

func (s *NotifierTestSuite) TestReleaseNotifier_MultipleSubscribers() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}
	subs := []domain.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(subs, nil)
	s.smtp.On("SendReleaseNotification", "alice@example.com", "token-alice", release).Return(nil)
	s.smtp.On("SendReleaseNotification", "bob@example.com", "token-bob", release).Return(nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}

func (s *NotifierTestSuite) TestReleaseNotifier_SMTPFails_ContinuesLoop() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}
	subs := []domain.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(subs, nil)
	s.smtp.On("SendReleaseNotification", "alice@example.com", "token-alice", release).
		Return(errors.New("smtp error"))
	s.smtp.On("SendReleaseNotification", "bob@example.com", "token-bob", release).
		Return(nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}
