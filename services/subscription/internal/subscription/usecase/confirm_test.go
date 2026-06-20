package usecase_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"

	"github.com/stretchr/testify/mock"
)

func (s *UseCaseSuite) TestConfirm_Success() {
	s.repo.On("Confirm", mock.Anything, "token123").Return(1, nil)

	err := s.confirm.Execute(context.Background(), "token123")

	s.NoError(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestConfirm_TokenNotFound() {
	s.repo.On("Confirm", mock.Anything, "token123").Return(0, db.ErrNotFound)

	err := s.confirm.Execute(context.Background(), "token123")

	s.ErrorIs(err, subscription.ErrTokenNotFound)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestConfirm_DBError() {
	s.repo.On("Confirm", mock.Anything, "token123").Return(0, errors.New("db error"))

	err := s.confirm.Execute(context.Background(), "token123")

	s.Error(err)
	s.assertExpectations()
}
