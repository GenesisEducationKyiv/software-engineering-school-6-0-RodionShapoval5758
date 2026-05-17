package service_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/service"
	"GithubReleaseNotificationAPI/internal/store"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_TokenNotFound() {
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(nil, store.ErrNotFound)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.ErrorIs(err, service.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_FindDBError() {
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(nil, errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_DeleteTokenNotFound() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(store.ErrNotFound)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.ErrorIs(err, service.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_DeleteDBError() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_OrphanCheckError() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.subRepo.On("HasAnyByRepositoryID", mock.Anything, int64(1)).Return(false, errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_RepositoryStillHasSubscribers() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.subRepo.On("HasAnyByRepositoryID", mock.Anything, int64(1)).Return(true, nil)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations() // repoRepo.DeleteByID has no .On() — panics if called
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_OrphanedRepositoryDeleted() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.subRepo.On("HasAnyByRepositoryID", mock.Anything, int64(1)).Return(false, nil)
	s.repoRepo.On("DeleteByID", mock.Anything, int64(1)).Return(nil)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_OrphanCleanup_RepoNotFound() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.subRepo.On("HasAnyByRepositoryID", mock.Anything, int64(1)).Return(false, nil)
	s.repoRepo.On("DeleteByID", mock.Anything, int64(1)).Return(store.ErrNotFound)

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestUnsubscribe_OrphanCleanup_DBError() {
	sub := &domain.Subscription{RepositoryID: 1}
	s.subRepo.On("FindByUnsubscribeToken", mock.Anything, "token123").Return(sub, nil)
	s.subRepo.On("DeleteByUnsubscribeToken", mock.Anything, "token123").Return(nil)
	s.subRepo.On("HasAnyByRepositoryID", mock.Anything, int64(1)).Return(false, nil)
	s.repoRepo.On("DeleteByID", mock.Anything, int64(1)).Return(errors.New("db error"))

	err := s.svc.Unsubscribe(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}
