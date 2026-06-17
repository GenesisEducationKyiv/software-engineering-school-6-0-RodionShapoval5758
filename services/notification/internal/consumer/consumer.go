package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/notification/internal/subscriber"

	"github.com/nats-io/nats.go/jetstream"
)

type Mailer interface {
	SendConfirmation(toEmail, repoName, confirmToken string) error
	SendRelease(toEmail, unsubscribeToken string, releaseTag, releaseName, releaseURL string) error
}

type SubscriberClient interface {
	ListConfirmed(ctx context.Context, repoID int64) ([]subscriber.Subscriber, error)
}

type Consumer struct {
	js     jetstream.JetStream
	mailer Mailer
	subs   SubscriberClient
}

func New(js jetstream.JetStream, m Mailer, subs SubscriberClient) *Consumer {
	return &Consumer{js: js, mailer: m, subs: subs}
}

func (c *Consumer) Start(ctx context.Context) error {
	cons, err := c.js.CreateOrUpdateConsumer(ctx, contract.StreamName, jetstream.ConsumerConfig{
		Durable:       "notification-consumer",
		FilterSubject: contract.SubjectAll,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		ack, term := processMessage(ctx, msg.Subject(), msg.Data(), c.mailer, c.subs)
		switch {
		case term:
			if err := msg.Term(); err != nil {
				slog.Error("msg term failed", "error", err)
			}
		case ack:
			if err := msg.Ack(); err != nil {
				slog.Error("msg ack failed", "error", err)
			}
		default:
			if err := msg.Nak(); err != nil {
				slog.Error("msg nak failed", "error", err)
			}
		}
	})
	if err != nil {
		return err
	}
	defer cc.Stop()

	<-ctx.Done()
	return nil
}

// processMessage decodes and dispatches a single message.
// Returns (ack=true, term=false) on success or unknown subject,
// (ack=false, term=true) on unmarshal failure,
// (ack=false, term=false) on transient failure (triggers Nak/redeliver).
func processMessage(ctx context.Context, subject string, data []byte, m Mailer, subs SubscriberClient) (ack bool, term bool) {
	switch subject {
	case contract.SubjectConfirmation:
		var ev contract.ConfirmationRequested
		if err := json.Unmarshal(data, &ev); err != nil {
			slog.Error("unmarshal confirmation event", "error", err)
			return false, true
		}
		if err := m.SendConfirmation(ev.Email, ev.RepoName, ev.ConfirmToken); err != nil {
			slog.Error("send confirmation email", "error", err, "email", ev.Email)
			return false, false
		}
		return true, false

	case contract.SubjectRelease:
		var ev contract.ReleaseDetected
		if err := json.Unmarshal(data, &ev); err != nil {
			slog.Error("unmarshal release event", "error", err)
			return false, true
		}

		confirmed, err := subs.ListConfirmed(ctx, ev.RepoID)
		if err != nil {
			slog.Error("fetch confirmed subscribers", "error", err, "repo_id", ev.RepoID)
			return false, false
		}

		for _, sub := range confirmed {
			if err := m.SendRelease(sub.Email, sub.UnsubscribeToken, ev.ReleaseTag, ev.ReleaseName, ev.ReleaseURL); err != nil {
				slog.Error("send release email", "error", err, "email", sub.Email)
			}
		}
		return true, false

	default:
		slog.Warn("unknown subject, skipping", "subject", subject)
		return true, false
	}
}
