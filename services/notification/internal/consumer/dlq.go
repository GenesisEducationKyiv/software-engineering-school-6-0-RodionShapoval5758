package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
)

func StartDLQInspector(ctx context.Context, js jetstream.JetStream) error {
	cons, err := js.CreateOrUpdateConsumer(ctx, contract.StreamDLQ, jetstream.ConsumerConfig{
		Durable:       "notification-dlq-consumer",
		FilterSubject: contract.SubjectDead,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		var dl contract.DeadLetter
		if err := json.Unmarshal(msg.Data(), &dl); err != nil {
			slog.Error("dlq inspector: unmarshal failed", "error", err)
			if err := msg.Ack(); err != nil {
				slog.Error("dlq inspector: ack failed", "error", err)
			}
			return
		}

		slog.Error("dead letter received",
			"original_subject", dl.OriginalSubject,
			"reason", dl.Reason,
			"attempts", dl.Attempts,
			"failed_at", dl.FailedAt,
		)

		if err := msg.Ack(); err != nil {
			slog.Error("dlq inspector: ack failed", "error", err)
		}
	})
	if err != nil {
		return err
	}
	defer cc.Stop()

	<-ctx.Done()

	return nil
}
