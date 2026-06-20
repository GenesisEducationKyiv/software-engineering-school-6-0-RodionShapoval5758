package subscription_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/transport/http/handler"
	"GithubReleaseNotificationAPI/internal/transport/http/respond"
	"GithubReleaseNotificationAPI/internal/transport/http/router"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite

	sub    *mockSubscriber
	conf   *mockConfirmer
	unsub  *mockUnsubscriber
	list   *mockLister
	router http.Handler
}

func (s *HandlerTestSuite) SetupTest() {
	s.sub = new(mockSubscriber)
	s.conf = new(mockConfirmer)
	s.unsub = new(mockUnsubscriber)
	s.list = new(mockLister)
	s.router = router.New(
		handler.New(s.sub, s.conf, s.unsub, s.list),
		"",
		metrics.New(prometheus.NewRegistry()),
		&stubPinger{},
		&stubPinger{},
	)
}

func (s *HandlerTestSuite) assertExpectations() {
	s.sub.AssertExpectations(s.T())
	s.conf.AssertExpectations(s.T())
	s.unsub.AssertExpectations(s.T())
	s.list.AssertExpectations(s.T())
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (s *HandlerTestSuite) performRequest(method, target, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *HandlerTestSuite) requireErrorResponse(rec *httptest.ResponseRecorder, status int) respond.ErrorResponse {
	s.Equal(status, rec.Code)

	var response respond.ErrorResponse
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &response))
	s.NotEmpty(response.ErrorMessage)

	return response
}

func (s *HandlerTestSuite) requireJSONResponse(rec *httptest.ResponseRecorder, status int, target any) {
	s.Equal(status, rec.Code)
	s.Equal("application/json", rec.Header().Get("Content-Type"))
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), target))
}
