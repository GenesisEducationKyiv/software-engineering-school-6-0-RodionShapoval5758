//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"strings"
)

func (s *IntegrationSuite) TestSubscribe_HappyPath() {
	body := `{"email":"user@example.com","repo":"owner/repo"}`
	w := s.do(http.MethodPost, "/api/subscribe", strings.NewReader(body))

	s.Equal(http.StatusOK, w.Code)

	var count int
	err := testPool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM subscriptions WHERE email = $1",
		"user@example.com",
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count)

	s.Equal(1, mailpitCount())
}
