package subscription_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"GithubReleaseNotificationAPI/services/subscription/internal/metrics"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/handler"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/respond"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/router"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
)

type stubKeys struct{ key *ecdsa.PublicKey }

func (s *stubKeys) Key() (crypto.PublicKey, error) { return s.key, nil }

type HandlerTestSuite struct {
	suite.Suite

	sub     *mockSubscriber
	conf    *mockConfirmer
	unsub   *mockUnsubscriber
	list    *mockLister
	signKey *ecdsa.PrivateKey
	router  http.Handler
}

func (s *HandlerTestSuite) SetupTest() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)
	s.signKey = key

	s.sub = new(mockSubscriber)
	s.conf = new(mockConfirmer)
	s.unsub = new(mockUnsubscriber)
	s.list = new(mockLister)
	s.router = router.New(
		handler.New(s.sub, s.conf, s.unsub, s.list),
		&stubKeys{&key.PublicKey},
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

func (s *HandlerTestSuite) newRequest(method, target, body, contentType string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req
}

func (s *HandlerTestSuite) serve(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *HandlerTestSuite) performRequest(method, target, body, contentType string) *httptest.ResponseRecorder {
	return s.serve(s.newRequest(method, target, body, contentType))
}

func (s *HandlerTestSuite) performAuthedRequest(method, target, body, contentType string) *httptest.ResponseRecorder {
	req := s.newRequest(method, target, body, contentType)
	req.Header.Set("Authorization", "Bearer "+s.authToken("user@example.com"))

	return s.serve(req)
}

func (s *HandlerTestSuite) authToken(email string) string {
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub":   "test-user",
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})

	signed, err := tok.SignedString(s.signKey)
	s.Require().NoError(err)

	return signed
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
