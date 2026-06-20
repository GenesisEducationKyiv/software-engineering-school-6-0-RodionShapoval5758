package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
)

const maxDeliver = 5

type outcome int

const (
	outcomeAck    outcome = iota
	outcomePoison         // permanent failure, retrying is pointless
	outcomeRetry          // transient failure, redeliver
)

type action int

const (
	actionAck action = iota
	actionNak
	actionDLQ
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
		MaxDeliver:    maxDeliver,
	})
	if err != nil {
		return err
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		meta, err := msg.Metadata()
		if err != nil {
			slog.Error("read msg metadata", "error", err)
			if err := msg.Nak(); err != nil {
				slog.Error("msg nak failed", "error", err)
			}
			return
		}

		oc, reason, sagaID := processMessage(msg.Subject(), msg.Data(), c.mailer)
		act := decideAction(oc, meta.NumDelivered)

		switch act {
		case actionDLQ:
			c.toDLQ(ctx, msg, reason, meta)
			if sagaID != "" {
				c.publishEmailFailed(ctx, sagaID, reason, meta)
			}
		case actionAck:
			if err := msg.Ack(); err != nil {
				slog.Error("msg ack failed", "error", err)
			}
			if sagaID != "" {
				c.publishEmailSent(ctx, sagaID, meta)
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

func decideAction(oc outcome, numDelivered uint64) action {
	switch oc {
	case outcomeAck:
		return actionAck
	case outcomePoison:
		return actionDLQ
	default:
		if numDelivered >= maxDeliver {
			return actionDLQ
		}
		return actionNak
	}
}

func processMessage(subject string, data []byte, m Mailer) (outcome, string, string) {
	switch subject {
	case contract.SubjectConfirmation:
		var ev contract.ConfirmationRequested
		if err := json.Unmarshal(data, &ev); err != nil {
			slog.Error("unmarshal confirmation event", "error", err)
			return outcomePoison, "unmarshalable", ""
		}
		if err := m.SendConfirmation(ev.Email, ev.RepoName, ev.ConfirmToken); err != nil {
			slog.Error("send confirmation email", "error", err, "email", ev.Email)
			return outcomeRetry, err.Error(), ev.SagaID
		}
		return outcomeAck, "", ev.SagaID

	case contract.SubjectRelease:
		var ev contract.ReleaseDetected
		if err := json.Unmarshal(data, &ev); err != nil {
			slog.Error("unmarshal release event", "error", err)
			return outcomePoison, "unmarshalable", ""
		}
		if err := m.SendRelease(ev.Email, ev.UnsubscribeToken, ev.ReleaseTag, ev.ReleaseName, ev.ReleaseURL); err != nil {
			slog.Error("send release email", "error", err, "email", ev.Email)
			return outcomeRetry, err.Error(), ""
		}
		return outcomeAck, "", ""

	default:
		slog.Warn("unknown subject, skipping", "subject", subject)
		return outcomeAck, "", ""
	}
}

func (c *Consumer) publishEmailSent(ctx context.Context, sagaID string, meta *jetstream.MsgMetadata) {
	data, err := json.Marshal(contract.EmailSent{SagaID: sagaID})
	if err != nil {
		slog.Error("marshal EmailSent", "error", err)
		return
	}
	msgID := "sent-" + strconv.FormatUint(meta.Sequence.Stream, 10)
	if _, err := c.js.Publish(ctx, contract.SubjectEmailSent, data, jetstream.WithMsgID(msgID)); err != nil {
		slog.Error("publish EmailSent failed", "saga_id", sagaID, "error", err)
	}
}

func (c *Consumer) publishEmailFailed(ctx context.Context, sagaID, reason string, meta *jetstream.MsgMetadata) {
	data, err := json.Marshal(contract.EmailFailed{SagaID: sagaID, Reason: reason})
	if err != nil {
		slog.Error("marshal EmailFailed", "error", err)
		return
	}
	msgID := "failed-" + strconv.FormatUint(meta.Sequence.Stream, 10)
	if _, err := c.js.Publish(ctx, contract.SubjectEmailFailed, data, jetstream.WithMsgID(msgID)); err != nil {
		slog.Error("publish EmailFailed failed", "saga_id", sagaID, "error", err)
	}
}

func (c *Consumer) toDLQ(ctx context.Context, msg jetstream.Msg, reason string, meta *jetstream.MsgMetadata) {
	dl := buildDeadLetter(msg.Subject(), msg.Data(), reason, meta.NumDelivered)

	data, err := json.Marshal(dl)
	if err != nil {
		slog.Error("marshal dead letter", "error", err)
		if err := msg.Nak(); err != nil {
			slog.Error("msg nak failed after marshal error", "error", err)
		}
		return
	}

	msgID := strconv.FormatUint(meta.Sequence.Stream, 10)
	if _, err := c.js.Publish(ctx, contract.SubjectDead, data, jetstream.WithMsgID(msgID)); err != nil {
		slog.Error("dlq publish failed", "error", err)
		if err := msg.Nak(); err != nil {
			slog.Error("msg nak failed after dlq error", "error", err)
		}
		return
	}

	if err := msg.Ack(); err != nil {
		slog.Error("msg ack failed after dlq publish", "error", err)
	}
}

func buildDeadLetter(subject string, data []byte, reason string, attempts uint64) contract.DeadLetter {
	payload := json.RawMessage(data)
	if !json.Valid(data) {
		// data is not JSON (e.g. binary or truly malformed); store as a JSON string so
		// the DeadLetter envelope can always be marshaled without error.
		encoded, _ := json.Marshal(string(data))
		payload = json.RawMessage(encoded)
	}

	return contract.DeadLetter{
		OriginalSubject: subject,
		Payload:         payload,
		Reason:          reason,
		Attempts:        attempts,
		FailedAt:        time.Now().UTC(),
	}
}
