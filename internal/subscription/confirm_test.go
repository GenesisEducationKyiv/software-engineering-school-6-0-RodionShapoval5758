package subscription_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/shared"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

func (s *ServiceTestSuite) TestConfirm_Success() {
	s.subRepo.On("Confirm", mock.Anything, "token123").Return(nil)

	err := s.svc.Confirm(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestConfirm_TokenNotFound() {
	s.subRepo.On("Confirm", mock.Anything, "token123").Return(shared.ErrNotFound)

	err := s.svc.Confirm(context.Background(), "token123")

	s.ErrorIs(err, subscription.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestConfirm_DBError() {
	s.subRepo.On("Confirm", mock.Anything, "token123").Return(errors.New("db error"))

	err := s.svc.Confirm(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}
