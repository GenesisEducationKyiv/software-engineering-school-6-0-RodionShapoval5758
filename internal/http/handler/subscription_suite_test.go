package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"GithubReleaseNotificationAPI/internal/http/handler"
	"GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/http/util"

	"github.com/stretchr/testify/suite"
)

type SubscriptionHandlerTestSuite struct {
	suite.Suite

	subscriptionService *mockSubscriptionService
	router              http.Handler
}

func (s *SubscriptionHandlerTestSuite) SetupTest() {
	s.subscriptionService = new(mockSubscriptionService)
	s.router = router.New(handler.New(s.subscriptionService), "")
}

func (s *SubscriptionHandlerTestSuite) assertExpectations() {
	s.subscriptionService.AssertExpectations(s.T())
}

func (s *SubscriptionHandlerTestSuite) performRequest(method, target, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *SubscriptionHandlerTestSuite) requireErrorResponse(rec *httptest.ResponseRecorder, status int) util.ErrorResponse {
	s.Equal(status, rec.Code)

	var response util.ErrorResponse
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &response))
	s.NotEmpty(response.ErrorMessage)

	return response
}

func (s *SubscriptionHandlerTestSuite) requireJSONResponse(rec *httptest.ResponseRecorder, status int, target any) {
	s.Equal(status, rec.Code)
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), target))
}

func TestSubscriptionHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionHandlerTestSuite))
}
