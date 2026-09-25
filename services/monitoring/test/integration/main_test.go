//go:build integration

package integration_test

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	testPool   *pgxpool.Pool
	testConn   *grpc.ClientConn
	stubServer *stubCatalogServer
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	if err := os.Chdir(root); err != nil {
		log.Fatalf("chdir to service root: %v", err)
	}

	ctx := context.Background()

	dsn, pgCleanup := resolvePostgres(ctx)
	defer pgCleanup()

	if err := db.RunMigrations(dsn); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("create db pool: %v", err)
	}
	testPool = pool
	defer testPool.Close()

	stubServer = &stubCatalogServer{}
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	grpcSrv := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(grpcSrv, stubServer)
	go func() { _ = grpcSrv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("create grpc conn: %v", err)
	}
	testConn = conn
	defer func() { _ = testConn.Close() }()

	return m.Run()
}

func resolvePostgres(ctx context.Context) (string, func()) {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn, func() {}
	}

	ctr, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("github_monitoring_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("password"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("get postgres connection string: %v", err)
	}

	return dsn, func() {
		if err := ctr.Terminate(ctx); err != nil {
			log.Printf("terminate postgres container: %v", err)
		}
	}
}

type stubCatalogServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	mu    sync.Mutex
	repos []*catalogv1.TrackedRepo
}

func (s *stubCatalogServer) setRepos(repos []*catalogv1.TrackedRepo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = repos
}

func (s *stubCatalogServer) ListTrackedRepos(_ context.Context, _ *catalogv1.ListTrackedReposRequest) (*catalogv1.ListTrackedReposResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &catalogv1.ListTrackedReposResponse{Repos: s.repos}, nil
}
