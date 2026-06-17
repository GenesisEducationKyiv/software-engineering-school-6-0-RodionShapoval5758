package subscription_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/subscription"
	"GithubReleaseNotificationAPI/internal/transport/http/handler"
	"GithubReleaseNotificationAPI/internal/transport/http/respond"
	"GithubReleaseNotificationAPI/internal/transport/http/router"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
)

type ServiceTestSuite struct {
	suite.Suite

	subRepo *mockSubscriptionRepository
	catalog *mockCatalogClient
	github  *mockGithubClient
	outbox  *mockOutboxWriter

	svc *subscription.Service
}

func (s *ServiceTestSuite) SetupTest() {
	s.subRepo = new(mockSubscriptionRepository)
	s.catalog = new(mockCatalogClient)
	s.github = new(mockGithubClient)
	s.outbox = new(mockOutboxWriter)

	s.svc = subscription.NewService(s.subRepo, s.catalog, s.github, s.outbox, &fakeTxBeginner{})
}

func (s *ServiceTestSuite) assertExpectations() {
	s.subRepo.AssertExpectations(s.T())
	s.catalog.AssertExpectations(s.T())
	s.github.AssertExpectations(s.T())
	s.outbox.AssertExpectations(s.T())
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type HandlerTestSuite struct {
	suite.Suite

	svc    *mockServiceForHandler
	router http.Handler
}

func (s *HandlerTestSuite) SetupTest() {
	s.svc = new(mockServiceForHandler)
	s.router = router.New(
		handler.New(s.svc),
		&stubInternalHandler{},
		"",
		metrics.New(prometheus.NewRegistry()),
		&stubPinger{},
		&stubPinger{},
	)
}

func (s *HandlerTestSuite) assertExpectations() {
	s.svc.AssertExpectations(s.T())
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

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
