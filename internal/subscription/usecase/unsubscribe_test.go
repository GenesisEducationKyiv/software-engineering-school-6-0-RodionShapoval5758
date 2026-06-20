package usecase_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

func (s *UseCaseSuite) TestUnsubscribe_TokenNotFound() {
	s.repo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(nil, db.ErrNotFound)

	err := s.unsubscribe.Execute(context.Background(), "token123")

	s.ErrorIs(err, subscription.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestUnsubscribe_FindDBError() {
	s.repo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(nil, errors.New("db error"))

	err := s.unsubscribe.Execute(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestUnsubscribe_DeleteTokenNotFound() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.repo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.repo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(db.ErrNotFound)

	err := s.unsubscribe.Execute(context.Background(), "token123")

	s.ErrorIs(err, subscription.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestUnsubscribe_DeleteDBError() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.repo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.repo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(errors.New("db error"))

	err := s.unsubscribe.Execute(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestUnsubscribe_CleanupSucceeds() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.repo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.repo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.catalog.On("DeleteIfOrphaned", mock.Anything, int64(1), mock.Anything).Return(nil)

	err := s.unsubscribe.Execute(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestUnsubscribe_CleanupError() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.repo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.repo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.catalog.On("DeleteIfOrphaned", mock.Anything, int64(1), mock.Anything).Return(errors.New("db error"))

	err := s.unsubscribe.Execute(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}
