package notifier

import (
	"context"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
)

// EnsureStream creates or updates the NOTIFICATIONS stream.
// The API service is the sole owner of stream configuration.
func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       contract.StreamName,
		Subjects:   []string{contract.SubjectAll},
		Storage:    jetstream.FileStorage,
		Duplicates: 2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	return nil
}
