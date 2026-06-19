package usecase_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

func (s *UseCaseSuite) TestListByEmail_InvalidEmail() {
	cases := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"malformed", "not-an-email"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()

			result, err := s.list.ByEmail(context.Background(), tc.email)

			s.ErrorIs(err, subscription.ErrInvalidEmailFormat)
			s.Nil(result)
			s.assertExpectations()
		})
	}
}

func (s *UseCaseSuite) TestListByEmail_RepoError() {
	s.repo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(nil, errors.New("db error"))

	result, err := s.list.ByEmail(context.Background(), "user@example.com")

	s.Error(err)
	s.Nil(result)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestListByEmail_Success() {
	subs := []subscription.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true},
	}
	s.repo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	result, err := s.list.ByEmail(context.Background(), "user@example.com")

	s.NoError(err)
	s.Equal(subs, result)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestListByEmail_NormalizesEmail() {
	subs := []subscription.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true},
	}
	s.repo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	result, err := s.list.ByEmail(context.Background(), "  user@example.com  ")

	s.NoError(err)
	s.Equal(subs, result)
	s.assertExpectations()
}
