package service_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/service"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionServiceTestSuite) TestListByEmail_InvalidEmail() {
	cases := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"malformed", "not-an-email"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			result, err := s.svc.ListByEmail(context.Background(), tc.email)

			s.ErrorIs(err, service.ErrInvalidEmailFormat)
			s.Nil(result)
			s.assertExpectations()
		})
	}
}

func (s *SubscriptionServiceTestSuite) TestListByEmail_RepoError() {
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(nil, errors.New("db error"))

	result, err := s.svc.ListByEmail(context.Background(), "user@example.com")

	s.Error(err)
	s.Nil(result)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestListByEmail_Success() {
	subs := []domain.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true},
	}
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	result, err := s.svc.ListByEmail(context.Background(), "user@example.com")

	s.NoError(err)
	s.Equal(subs, result)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestListByEmail_EmptyList() {
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return([]domain.SubscriptionDetails{}, nil)

	result, err := s.svc.ListByEmail(context.Background(), "user@example.com")

	s.NoError(err)
	s.Empty(result)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestListByEmail_NormalizesEmail() {
	subs := []domain.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true},
	}
	// Mock registered with trimmed email — if the service passes the raw padded
	// value, the expectation won't match and AssertExpectations will fail.
	s.subRepo.On("ListSubscriptionDetailsByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	result, err := s.svc.ListByEmail(context.Background(), "  user@example.com  ")

	s.NoError(err)
	s.Equal(subs, result)
	s.assertExpectations()
}
