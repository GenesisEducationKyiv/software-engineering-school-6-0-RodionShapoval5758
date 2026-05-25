package handler_test

import (
	"errors"
	"net/http"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/http/models"
	"GithubReleaseNotificationAPI/internal/service"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionHandlerTestSuite) TestListSubscriptions_HappyPath() {
	subscriptions := []domain.SubscriptionDetails{
		{
			Email:       "user@example.com",
			Repo:        "owner/repo",
			Confirmed:   true,
			LastSeenTag: new("v1.0.0"),
		},
		{
			Email:     "user@example.com",
			Repo:      "owner/other",
			Confirmed: false,
		},
	}

	s.subscriptionService.
		On("ListByEmail", mock.Anything, "user@example.com").
		Return(subscriptions, nil)

	rec := s.performRequest(http.MethodGet, "/api/subscriptions?email=user%40example.com", "", "")

	var response []models.SubscriptionResponse
	s.requireJSONResponse(rec, http.StatusOK, &response)

	s.Require().Len(response, 2)
	s.Equal("user@example.com", response[0].Email)
	s.Equal("owner/repo", response[0].Repo)
	s.True(response[0].Confirmed)
	s.Equal("v1.0.0", response[0].LastSeenTag)
	s.Equal("not available yet", response[1].LastSeenTag)
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestListSubscriptions_MissingEmail() {
	rec := s.performRequest(http.MethodGet, "/api/subscriptions", "", "")

	s.requireErrorResponse(rec, http.StatusBadRequest)
	s.subscriptionService.AssertNotCalled(s.T(), "ListByEmail", mock.Anything, mock.Anything)
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestListSubscriptions_EmptyList() {
	s.subscriptionService.
		On("ListByEmail", mock.Anything, "user@example.com").
		Return([]domain.SubscriptionDetails{}, nil)

	rec := s.performRequest(http.MethodGet, "/api/subscriptions?email=user%40example.com", "", "")

	var response []models.SubscriptionResponse
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.NotNil(response)
	s.Empty(response)
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestListSubscriptions_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid email", service.ErrInvalidEmailFormat, http.StatusBadRequest},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.subscriptionService.
				On("ListByEmail", mock.Anything, "user@example.com").
				Return(nil, tc.err)

			rec := s.performRequest(http.MethodGet, "/api/subscriptions?email=user%40example.com", "", "")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}
