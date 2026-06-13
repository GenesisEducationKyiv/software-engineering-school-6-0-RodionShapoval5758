package subscription_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/shared"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

func (s *ServiceTestSuite) TestUnsubscribe_TokenNotFound() {
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(nil, shared.ErrNotFound)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.ErrorIs(err, subscription.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestUnsubscribe_FindDBError() {
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(nil, errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestUnsubscribe_DeleteTokenNotFound() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(shared.ErrNotFound)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.ErrorIs(err, subscription.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestUnsubscribe_DeleteDBError() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestUnsubscribe_CleanupSucceeds() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.catalog.On("DeleteIfOrphaned", mock.Anything, int64(1), mock.Anything).Return(nil)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestUnsubscribe_CleanupError() {
	sub := &subscription.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.catalog.On("DeleteIfOrphaned", mock.Anything, int64(1), mock.Anything).Return(errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}
