package handler_test

import (
	"errors"
	"net/http"

	"GithubReleaseNotificationAPI/internal/service"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionHandlerTestSuite) TestUnsubscribe_HappyPath() {
	s.subscriptionService.
		On("Unsubscribe", mock.Anything, "unsubscribe-token").
		Return(nil)

	rec := s.performRequest(http.MethodGet, "/api/unsubscribe/unsubscribe-token", "", "")

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Unsubscribed successfully", response["message"])
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestUnsubscribe_ShortToken() {
	rec := s.performRequest(http.MethodGet, "/api/unsubscribe/short", "", "")

	s.requireErrorResponse(rec, http.StatusBadRequest)
	s.subscriptionService.AssertNotCalled(s.T(), "Unsubscribe", mock.Anything, mock.Anything)
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestUnsubscribe_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"token not found", service.ErrTokenNotFound, http.StatusNotFound},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.subscriptionService.
				On("Unsubscribe", mock.Anything, "unsubscribe-token").
				Return(tc.err)

			rec := s.performRequest(http.MethodGet, "/api/unsubscribe/unsubscribe-token", "", "")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}
