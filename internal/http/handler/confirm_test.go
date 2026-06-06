package handler_test

import (
	"errors"
	"net/http"

	"GithubReleaseNotificationAPI/internal/service"

	"github.com/stretchr/testify/mock"
)

func (s *SubscriptionHandlerTestSuite) TestConfirm_HappyPath() {
	s.subscriptionService.
		On("Confirm", mock.Anything, "confirm-token").
		Return(nil)

	rec := s.performRequest(http.MethodGet, "/api/confirm/confirm-token", "", "")

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Subscription confirmed successfully", response["message"])
	s.assertExpectations()
}

func (s *SubscriptionHandlerTestSuite) TestConfirm_ServiceErrors() {
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
				On("Confirm", mock.Anything, "confirm-token").
				Return(tc.err)

			rec := s.performRequest(http.MethodGet, "/api/confirm/confirm-token", "", "")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}
