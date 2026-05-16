package service_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/service"
	"GithubReleaseNotificationAPI/internal/store"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionServiceTestSuite) TestConfirm_Success() {
	s.subRepo.On("Confirm", mock.Anything, "token123").Return(nil)

	err := s.svc.Confirm(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestConfirm_TokenNotFound() {
	s.subRepo.On("Confirm", mock.Anything, "token123").Return(store.ErrNotFound)

	err := s.svc.Confirm(context.Background(), "token123")

	s.ErrorIs(err, service.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestConfirm_DBError() {
	s.subRepo.On("Confirm", mock.Anything, "token123").Return(errors.New("db error"))

	err := s.svc.Confirm(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}
