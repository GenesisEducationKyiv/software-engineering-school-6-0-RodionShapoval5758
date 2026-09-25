package saga

import (
	"context"
	"encoding/json"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
)

type Consumer struct {
	js           jetstream.JetStream
	orchestrator *Orchestrator
}

func NewConsumer(js jetstream.JetStream, o *Orchestrator) *Consumer {
	return &Consumer{js: js, orchestrator: o}
}

func (c *Consumer) Start(ctx context.Context) error {
	cons, err := c.js.CreateOrUpdateConsumer(ctx, contract.StreamSaga, jetstream.ConsumerConfig{
		Durable:       "saga-reply-consumer",
		FilterSubject: contract.SubjectSagaAll,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err := c.handle(ctx, msg); err != nil {
			slog.Error("saga consumer handle error", "subject", msg.Subject(), "error", err)
			if err := msg.Nak(); err != nil {
				slog.Error("saga consumer nak failed", "error", err)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("saga consumer ack failed", "error", err)
		}
	})
	if err != nil {
		return err
	}
	defer cc.Stop()

	<-ctx.Done()

	return nil
}

func (c *Consumer) handle(ctx context.Context, msg jetstream.Msg) error {
	switch msg.Subject() {
	case contract.SubjectEmailSent:
		var ev contract.EmailSent
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			slog.Error("saga: unmarshal EmailSent", "error", err)
			return nil
		}

		return c.orchestrator.HandleEmailSent(ctx, ev.SagaID)

	case contract.SubjectEmailFailed:
		var ev contract.EmailFailed
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			slog.Error("saga: unmarshal EmailFailed", "error", err)
			return nil
		}

		return c.orchestrator.HandleEmailFailed(ctx, ev.SagaID, ev.Reason)

	default:
		slog.Warn("saga consumer: unknown subject", "subject", msg.Subject())
		return nil
	}
}
