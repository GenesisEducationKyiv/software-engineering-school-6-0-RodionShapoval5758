//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	httpHandler "GithubReleaseNotificationAPI/internal/http/handler"
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/mail"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/service"
	"GithubReleaseNotificationAPI/internal/store"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/suite"
)

const testAPIKey = "test-integration-key"

type IntegrationSuite struct {
	suite.Suite
	router     http.Handler
	githubFake *fakeGithubClient
}

func (s *IntegrationSuite) SetupSuite() {
	subRepo := store.NewSubscriptionRepository(testPool)
	repoRepo := store.NewRepoRepository(testPool)
	smtpClient := mail.NewSMTPService(smtpHost, smtpPort, "", "", "noreply@localhost", "http://localhost:8080")
	s.githubFake = &fakeGithubClient{}
	svc := service.NewSubscriptionService(subRepo, repoRepo, s.githubFake, smtpClient)
	h := httpHandler.New(svc)
	s.router = httpRouter.New(h, testAPIKey, metrics.New(prometheus.NewRegistry()))
}

func (s *IntegrationSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(), "TRUNCATE subscriptions, repositories CASCADE")
	s.Require().NoError(err)
	clearMailpit()
	s.githubFake.err = nil
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}
