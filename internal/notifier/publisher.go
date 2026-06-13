package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"GithubReleaseNotificationAPI/contract"

	"github.com/nats-io/nats.go/jetstream"
)

type msgPublisher interface {
	Publish(ctx context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
}

type Publisher struct {
	pub msgPublisher
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{pub: js}
}

func (p *Publisher) SendConfirmation(toEmail, repoName, confirmToken string) error {
	payload, err := json.Marshal(contract.ConfirmationRequested{
		Email:        toEmail,
		RepoName:     repoName,
		ConfirmToken: confirmToken,
	})
	if err != nil {
		return fmt.Errorf("marshal confirmation: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := p.pub.Publish(ctx, contract.SubjectConfirmation, payload); err != nil {
		return fmt.Errorf("publish confirmation: %w", err)
	}

	return nil
}

func (p *Publisher) SendReleaseEmails(recipients []ReleaseRecipient, release ReleaseInfo) error {
	var errs []error

	for _, r := range recipients {
		payload, err := json.Marshal(contract.ReleasePublished{
			Email:            r.Email,
			UnsubscribeToken: r.UnsubscribeToken,
			ReleaseTag:       release.Tag,
			ReleaseName:      release.Name,
			ReleaseURL:       release.URL,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("marshal release for %s: %w", r.Email, err))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, pubErr := p.pub.Publish(ctx, contract.SubjectRelease, payload)
		cancel()

		if pubErr != nil {
			errs = append(errs, fmt.Errorf("publish release to %s: %w", r.Email, pubErr))
		}
	}

	return errors.Join(errs...)
}

func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     contract.StreamName,
		Subjects: []string{contract.SubjectAll},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	return nil
}
