//go:build integration

package integration_test

import (
	"net/http"
)

func (s *IntegrationSuite) TestAuth_MissingHeader() {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/subscribe"},
		{http.MethodGet, "/api/subscriptions"},
	}

	for _, tc := range cases {
		s.Run(tc.method+" "+tc.path, func() {
			w := s.doWithAuth(tc.method, tc.path, nil, "")
			s.Equal(http.StatusUnauthorized, w.Code)
		})
	}
}

func (s *IntegrationSuite) TestAuth_WrongKey() {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/subscribe"},
		{http.MethodGet, "/api/subscriptions"},
	}

	for _, tc := range cases {
		s.Run(tc.method+" "+tc.path, func() {
			w := s.doWithAuth(tc.method, tc.path, nil, "Bearer wrong-key")
			s.Equal(http.StatusUnauthorized, w.Code)
		})
	}
}

func (s *IntegrationSuite) TestAuth_UnprotectedRoutes_NoAuthRequired() {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/confirm/sometoken"},
		{http.MethodGet, "/api/unsubscribe/sometoken"},
	}

	for _, tc := range cases {
		s.Run(tc.method+" "+tc.path, func() {
			w := s.doWithAuth(tc.method, tc.path, nil, "")
			s.NotEqual(http.StatusUnauthorized, w.Code)
		})
	}
}
