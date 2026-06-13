package subscription_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

func (s *ServiceTestSuite) TestListByEmail_InvalidEmail() {
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

			result, err := s.svc.ListByEmail(context.Background(), tc.email)

			s.ErrorIs(err, subscription.ErrInvalidEmailFormat)
			s.Nil(result)
			s.assertExpectations()
		})
	}
}

func (s *ServiceTestSuite) TestListByEmail_RepoError() {
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(nil, errors.New("db error"))

	result, err := s.svc.ListByEmail(context.Background(), "user@example.com")

	s.Error(err)
	s.Nil(result)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestListByEmail_Success() {
	subs := []subscription.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true},
	}
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	result, err := s.svc.ListByEmail(context.Background(), "user@example.com")

	s.NoError(err)
	s.Equal(subs, result)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestListByEmail_NormalizesEmail() {
	subs := []subscription.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true},
	}
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	result, err := s.svc.ListByEmail(context.Background(), "  user@example.com  ")

	s.NoError(err)
	s.Equal(subs, result)
	s.assertExpectations()
}
