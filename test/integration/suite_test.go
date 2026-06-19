//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/metrics"
	"GithubReleaseNotificationAPI/internal/outbox"
	"GithubReleaseNotificationAPI/internal/subscription"
	"GithubReleaseNotificationAPI/internal/subscription/usecase"
	"GithubReleaseNotificationAPI/internal/transport/http/handler"
	httpRouter "GithubReleaseNotificationAPI/internal/transport/http/router"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/suite"
)

type noopPinger struct{}

func (noopPinger) Ping(context.Context) error { return nil }

const testAPIKey = "test-integration-key"

type IntegrationSuite struct {
	suite.Suite
	router     http.Handler
	githubFake *fakeGithubClient
}

func (s *IntegrationSuite) SetupSuite() {
	subRepo := subscription.NewRepository(testPool)
	outboxStore := outbox.NewStore()
	ensureCat := catalog.NewEnsure(testPool)
	deleteCat := catalog.NewDeleteIfOrphaned(testPool)
	s.githubFake = &fakeGithubClient{}

	sub := usecase.NewSubscribe(subRepo, ensureCat, s.githubFake, outboxStore, db.WrapPool(testPool))
	conf := usecase.NewConfirm(subRepo)
	unsub := usecase.NewUnsubscribe(subRepo, deleteCat)
	list := usecase.NewList(subRepo)

	h := handler.New(sub, conf, unsub, list)
	m := metrics.New(prometheus.NewRegistry())
	s.router = httpRouter.New(h, testAPIKey, m, noopPinger{}, noopPinger{})
}

func (s *IntegrationSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(), "TRUNCATE subscriptions, repositories CASCADE")
	s.Require().NoError(err)
	s.githubFake.err = nil
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}
