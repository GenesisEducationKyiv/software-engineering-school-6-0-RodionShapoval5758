//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"GithubReleaseNotificationAPI/internal/catalog"
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/subscription"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/suite"
)

type noopNotifier struct{}

func (n *noopNotifier) SendConfirmation(toEmail, repoName, confirmToken string) error { return nil }

const testAPIKey = "test-integration-key"

type IntegrationSuite struct {
	suite.Suite
	router     http.Handler
	githubFake *fakeGithubClient
}

func (s *IntegrationSuite) SetupSuite() {
	subRepo := subscription.NewRepository(testPool)
	catalogSvc := catalog.New(testPool)
	s.githubFake = &fakeGithubClient{}
	svc := subscription.NewService(subRepo, catalogSvc, s.githubFake, &noopNotifier{})
	h := subscription.New(svc)
	s.router = httpRouter.New(h, testAPIKey, metrics.New(prometheus.NewRegistry()))
}

func (s *IntegrationSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(), "TRUNCATE subscriptions, repositories CASCADE")
	s.Require().NoError(err)
	s.githubFake.err = nil
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}
