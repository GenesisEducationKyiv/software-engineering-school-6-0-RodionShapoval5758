package handler_test

import (
	"errors"
	"net/http"

	"GithubReleaseNotificationAPI/internal/service"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionHandlerTestSuite) TestSubscribe_JSONHappyPath() {
	s.subscriptionService.
		On("Subscribe", mock.Anything, "user@example.com", "owner/repo").
		Return(nil)

	rec := s.performRequest(
		http.MethodPost,
		"/api/subscribe",
		`{"email":"user@example.com","repo":"owner/repo"}`,
		"application/json",
	)

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Subscription successful. Confirmation email sent", response["message"])
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestSubscribe_FormHappyPath() {
	s.subscriptionService.
		On("Subscribe", mock.Anything, "user@example.com", "owner/repo").
		Return(nil)

	rec := s.performRequest(
		http.MethodPost,
		"/api/subscribe",
		"email=user%40example.com&repo=owner%2Frepo",
		"application/x-www-form-urlencoded",
	)

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Subscription successful. Confirmation email sent", response["message"])
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestSubscribe_BadRequestBeforeService() {
	cases := []struct {
		name        string
		body        string
		contentType string
	}{
		{
			name:        "malformed json",
			body:        `{"email":"user@example.com","repo":`,
			contentType: "application/json",
		},
		{
			name:        "empty email",
			body:        `{"email":"","repo":"owner/repo"}`,
			contentType: "application/json",
		},
		{
			name:        "empty repo",
			body:        `{"email":"user@example.com","repo":""}`,
			contentType: "application/json",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()

			rec := s.performRequest(http.MethodPost, "/api/subscribe", tc.body, tc.contentType)

			s.requireErrorResponse(rec, http.StatusBadRequest)
			s.subscriptionService.AssertNotCalled(s.T(), "Subscribe", mock.Anything, mock.Anything, mock.Anything)
			s.assertExpectations()
		})
	}
}

func (s *SubscriptionHandlerTestSuite) TestSubscribe_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid email", service.ErrInvalidEmailFormat, http.StatusBadRequest},
		{"invalid repo", service.ErrInvalidRepoFormat, http.StatusBadRequest},
		{"repo not found", service.ErrRepoNotFound, http.StatusNotFound},
		{"already exists", service.ErrSubscriptionAlreadyExists, http.StatusConflict},
		{"rate limited", service.ErrTooMuchRequests, http.StatusTooManyRequests},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.subscriptionService.
				On("Subscribe", mock.Anything, "user@example.com", "owner/repo").
				Return(tc.err)

			rec := s.performRequest(
				http.MethodPost,
				"/api/subscribe",
				`{"email":"user@example.com","repo":"owner/repo"}`,
				"application/json",
			)

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}
