package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
)

type Mailer interface {
	SendConfirmation(toEmail, repoName, confirmToken string) error
	SendRelease(toEmail, unsubscribeToken string, releaseTag, releaseName, releaseURL string) error
}

type Consumer struct {
	js     jetstream.JetStream
	mailer Mailer
}

func New(js jetstream.JetStream, m Mailer) *Consumer {
	return &Consumer{js: js, mailer: m}
}

func (c *Consumer) Start(ctx context.Context) error {
	cons, err := c.js.CreateOrUpdateConsumer(ctx, contract.StreamName, jetstream.ConsumerConfig{
		Durable:       "notification-consumer",
		FilterSubject: contract.SubjectAll,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		ack, term := processMessage(msg.Subject(), msg.Data(), c.mailer)
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
// (ack=false, term=false) on send failure (triggers Nak/redeliver).
func processMessage(subject string, data []byte, m Mailer) (ack bool, term bool) {
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
		var ev contract.ReleasePublished
		if err := json.Unmarshal(data, &ev); err != nil {
			slog.Error("unmarshal release event", "error", err)
			return false, true
		}
		if err := m.SendRelease(ev.Email, ev.UnsubscribeToken, ev.ReleaseTag, ev.ReleaseName, ev.ReleaseURL); err != nil {
			slog.Error("send release email", "error", err, "email", ev.Email)
			return false, false
		}
		return true, false

	default:
		slog.Warn("unknown subject, skipping", "subject", subject)
		return true, false
	}
}
