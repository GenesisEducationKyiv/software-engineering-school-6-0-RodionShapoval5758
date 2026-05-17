//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	httpHandler "GithubReleaseNotificationAPI/internal/http/handler"
	httpRouter "GithubReleaseNotificationAPI/internal/http/router"
	"GithubReleaseNotificationAPI/internal/mail"
	"GithubReleaseNotificationAPI/internal/service"
	repoStore "GithubReleaseNotificationAPI/internal/store/repository"
	subStore "GithubReleaseNotificationAPI/internal/store/subscription"

	"github.com/stretchr/testify/suite"
)

const testAPIKey = "test-integration-key"

type IntegrationSuite struct {
	suite.Suite
	router http.Handler
}

func (s *IntegrationSuite) SetupSuite() {
	subRepo := subStore.NewSubscriptionRepository(testPool)
	repoRepo := repoStore.NewRepositoryRepository(testPool)
	smtpClient := mail.NewSMTPService("localhost", "1025", "", "", "noreply@localhost", "http://localhost:8080")
	svc := service.NewSubscriptionService(subRepo, repoRepo, &fakeGithubClient{}, smtpClient)
	h := httpHandler.New(svc)
	s.router = httpRouter.New(h, testAPIKey)
}

func (s *IntegrationSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(), "TRUNCATE subscriptions, repositories CASCADE")
	s.Require().NoError(err)
	clearMailpit()
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}
