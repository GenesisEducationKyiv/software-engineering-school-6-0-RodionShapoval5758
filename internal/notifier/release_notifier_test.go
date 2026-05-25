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

func (s *NotifierTestSuite) TestReleaseNotifier_MultipleSubscribers() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}
	subs := []domain.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(subs, nil)
	s.smtp.On("SendReleaseNotifications", subs, release).Return(nil)

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.NoError(err)
	s.assertExpectations()
}

func (s *NotifierTestSuite) TestReleaseNotifier_SMTPError_Propagates() {
	repo := domain.Repository{ID: 1, FullName: "owner/repo"}
	release := &domain.Release{Tag: "v1.0.0"}
	subs := []domain.Subscription{
		{Email: "alice@example.com", UnsubscribeToken: "token-alice"},
		{Email: "bob@example.com", UnsubscribeToken: "token-bob"},
	}

	s.subRepo.On("ListConfirmedByRepositoryID", mock.Anything, int64(1)).
		Return(subs, nil)
	s.smtp.On("SendReleaseNotifications", subs, release).
		Return(errors.New("partial smtp failure"))

	err := s.releaseNotifier.NotifyConfirmedSubscribers(context.Background(), repo, release)

	s.Error(err)
	s.assertExpectations()
}
