package subscription_test

import (
	"errors"
	"net/http"

	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/stretchr/testify/mock"
)

type subscriptionResp struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`
}

func (s *HandlerTestSuite) TestSubscribe_JSONHappyPath() {
	s.svc.On("Subscribe", mock.Anything, "user@example.com", "owner/repo").Return(nil)

	rec := s.performRequest(http.MethodPost, "/api/subscribe", `{"email":"user@example.com","repo":"owner/repo"}`, "application/json")

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Subscription successful. Confirmation email sent", response["message"])
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestSubscribe_FormHappyPath() {
	s.svc.On("Subscribe", mock.Anything, "user@example.com", "owner/repo").Return(nil)

	rec := s.performRequest(http.MethodPost, "/api/subscribe", "email=user%40example.com&repo=owner%2Frepo", "application/x-www-form-urlencoded")

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Subscription successful. Confirmation email sent", response["message"])
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestSubscribe_BadRequestBeforeService() {
	cases := []struct {
		name        string
		body        string
		contentType string
	}{
		{"malformed json", `{"email":"user@example.com","repo":`, "application/json"},
		{"empty email", `{"email":"","repo":"owner/repo"}`, "application/json"},
		{"empty repo", `{"email":"user@example.com","repo":""}`, "application/json"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			rec := s.performRequest(http.MethodPost, "/api/subscribe", tc.body, tc.contentType)
			s.requireErrorResponse(rec, http.StatusBadRequest)
			s.svc.AssertNotCalled(s.T(), "Subscribe", mock.Anything, mock.Anything, mock.Anything)
			s.assertExpectations()
		})
	}
}

func (s *HandlerTestSuite) TestSubscribe_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid email", subscription.ErrInvalidEmailFormat, http.StatusBadRequest},
		{"invalid repo", subscription.ErrInvalidRepoFormat, http.StatusBadRequest},
		{"repo not found", subscription.ErrRepoNotFound, http.StatusNotFound},
		{"already exists", subscription.ErrSubscriptionAlreadyExists, http.StatusConflict},
		{"rate limited", subscription.ErrTooMuchRequests, http.StatusTooManyRequests},
		{"github unauthorized", subscription.ErrGitHubUnauthorized, http.StatusBadGateway},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.svc.On("Subscribe", mock.Anything, "user@example.com", "owner/repo").Return(tc.err)

			rec := s.performRequest(http.MethodPost, "/api/subscribe", `{"email":"user@example.com","repo":"owner/repo"}`, "application/json")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}

func (s *HandlerTestSuite) TestConfirm_HappyPath() {
	s.svc.On("Confirm", mock.Anything, "confirm-token").Return(nil)

	rec := s.performRequest(http.MethodGet, "/api/confirm/confirm-token", "", "")

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Subscription confirmed successfully", response["message"])
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestConfirm_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"token not found", subscription.ErrTokenNotFound, http.StatusNotFound},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.svc.On("Confirm", mock.Anything, "confirm-token").Return(tc.err)

			rec := s.performRequest(http.MethodGet, "/api/confirm/confirm-token", "", "")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}

func (s *HandlerTestSuite) TestUnsubscribe_HappyPath() {
	s.svc.On("Unsubscribe", mock.Anything, "unsubscribe-token").Return(nil)

	rec := s.performRequest(http.MethodGet, "/api/unsubscribe/unsubscribe-token", "", "")

	var response map[string]string
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Equal("Unsubscribed successfully", response["message"])
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestUnsubscribe_ShortToken() {
	rec := s.performRequest(http.MethodGet, "/api/unsubscribe/short", "", "")

	s.requireErrorResponse(rec, http.StatusBadRequest)
	s.svc.AssertNotCalled(s.T(), "Unsubscribe", mock.Anything, mock.Anything)
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestUnsubscribe_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"token not found", subscription.ErrTokenNotFound, http.StatusNotFound},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.svc.On("Unsubscribe", mock.Anything, "unsubscribe-token").Return(tc.err)

			rec := s.performRequest(http.MethodGet, "/api/unsubscribe/unsubscribe-token", "", "")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}

func (s *HandlerTestSuite) TestListSubscriptions_HappyPath() {
	subs := []subscription.SubscriptionDetails{
		{Email: "user@example.com", Repo: "owner/repo", Confirmed: true, LastSeenTag: "v1.0.0"},
		{Email: "user@example.com", Repo: "owner/other", Confirmed: false},
	}
	s.svc.On("ListByEmail", mock.Anything, "user@example.com").Return(subs, nil)

	rec := s.performRequest(http.MethodGet, "/api/subscriptions?email=user%40example.com", "", "")

	var response []subscriptionResp
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.Require().Len(response, 2)
	s.Equal("user@example.com", response[0].Email)
	s.Equal("owner/repo", response[0].Repo)
	s.True(response[0].Confirmed)
	s.Equal("v1.0.0", response[0].LastSeenTag)
	s.Equal("not available yet", response[1].LastSeenTag)
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestListSubscriptions_MissingEmail() {
	rec := s.performRequest(http.MethodGet, "/api/subscriptions", "", "")

	s.requireErrorResponse(rec, http.StatusBadRequest)
	s.svc.AssertNotCalled(s.T(), "ListByEmail", mock.Anything, mock.Anything)
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestListSubscriptions_EmptyList() {
	s.svc.On("ListByEmail", mock.Anything, "user@example.com").Return([]subscription.SubscriptionDetails{}, nil)

	rec := s.performRequest(http.MethodGet, "/api/subscriptions?email=user%40example.com", "", "")

	var response []subscriptionResp
	s.requireJSONResponse(rec, http.StatusOK, &response)
	s.NotNil(response)
	s.Empty(response)
	s.assertExpectations()
}

func (s *HandlerTestSuite) TestListSubscriptions_ServiceErrors() {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid email", subscription.ErrInvalidEmailFormat, http.StatusBadRequest},
		{"unknown error", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			s.svc.On("ListByEmail", mock.Anything, "user@example.com").Return(nil, tc.err)

			rec := s.performRequest(http.MethodGet, "/api/subscriptions?email=user%40example.com", "", "")

			s.requireErrorResponse(rec, tc.wantStatus)
			s.assertExpectations()
		})
	}
}
