//go:build integration

package integration_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	"GithubReleaseNotificationAPI/services/subscription/internal/metrics"
	"GithubReleaseNotificationAPI/services/subscription/internal/outbox"
	"GithubReleaseNotificationAPI/services/subscription/internal/saga"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription/usecase"
	grpchandler "GithubReleaseNotificationAPI/services/subscription/internal/transport/grpc/handler"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/handler"
	httpRouter "GithubReleaseNotificationAPI/services/subscription/internal/transport/http/router"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type noopPinger struct{}

func (noopPinger) Ping(context.Context) error { return nil }

const testAPIKey = "test-integration-key"

type IntegrationSuite struct {
	suite.Suite
	router     http.Handler
	githubFake *fakeGithubClient
	grpcConn   *grpc.ClientConn
}

func (s *IntegrationSuite) SetupSuite() {
	subRepo := subscription.NewRepository(testPool)
	outboxStore := outbox.NewStore()
	ensureCat := catalog.NewEnsure(testPool)
	deleteCat := catalog.NewDeleteIfOrphaned(testPool)
	s.githubFake = &fakeGithubClient{}

	sagaStore := saga.NewStore()
	sagaOrchestrator := saga.NewOrchestrator(sagaStore, db.WrapPool(testPool), deleteCat, subRepo)

	sub := usecase.NewSubscribe(subRepo, ensureCat, s.githubFake, outboxStore, sagaStore, db.WrapPool(testPool), 24*time.Hour)
	conf := usecase.NewConfirm(subRepo, sagaOrchestrator)
	unsub := usecase.NewUnsubscribe(subRepo, deleteCat)
	list := usecase.NewList(subRepo)

	h := handler.New(sub, conf, unsub, list)
	m := metrics.New(prometheus.NewRegistry())
	s.router = httpRouter.New(h, testAPIKey, m, noopPinger{}, noopPinger{})

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcSrv := grpc.NewServer()
	listTrackedUC := catalog.NewListTracked(testPool)
	catalogv1.RegisterCatalogServiceServer(grpcSrv, grpchandler.NewCatalog(listTrackedUC))

	go func() { _ = grpcSrv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	s.grpcConn = conn
}

func (s *IntegrationSuite) TearDownSuite() {
	if s.grpcConn != nil {
		_ = s.grpcConn.Close()
	}
}

func (s *IntegrationSuite) SetupTest() {
	_, err := testPool.Exec(context.Background(), "TRUNCATE subscribe_sagas, subscriptions, repositories CASCADE")
	s.Require().NoError(err)
	s.githubFake.err = nil
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}
