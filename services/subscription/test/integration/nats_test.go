//go:build integration

package integration_test

import (
	"context"
	"log"
	"os"
	"time"

	"GithubReleaseNotificationAPI/contract"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	tcnats "github.com/testcontainers/testcontainers-go/modules/nats"
)

var (
	testNC *natsgo.Conn
	testJS jetstream.JetStream
)

func resolveNATS(ctx context.Context) (string, func()) {
	if url := os.Getenv("TEST_NATS_URL"); url != "" {
		return url, func() {}
	}

	ctr, err := tcnats.Run(ctx, "nats:2-alpine")
	if err != nil {
		log.Fatalf("start nats container: %v", err)
	}

	url, err := ctr.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get nats connection string: %v", err)
	}

	return url, func() {
		if err := ctr.Terminate(ctx); err != nil {
			log.Printf("terminate nats container: %v", err)
		}
	}
}

func connectNATS(url string) (*natsgo.Conn, jetstream.JetStream) {
	nc, err := natsgo.Connect(url, natsgo.Name("integration-test"))
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("init jetstream: %v", err)
	}

	return nc, js
}

func ensureNotificationStream(ctx context.Context, js jetstream.JetStream) {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       contract.StreamName,
		Subjects:   []string{contract.SubjectAll},
		Storage:    jetstream.MemoryStorage,
		Retention:  jetstream.WorkQueuePolicy,
		Duplicates: 2 * time.Minute,
		MaxAge:     24 * time.Hour,
		MaxBytes:   512 * 1024 * 1024,
	})
	if err != nil {
		log.Fatalf("ensure notification stream: %v", err)
	}
}
