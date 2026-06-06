package monitoring

import (
	"context"
	"log/slog"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/notification"
)

type ReleaseNotifier struct {
	mailer           mailer
	subscriberReader subscriberReader
}

func NewReleaseNotifier(mailer mailer, subscriberReader subscriberReader) *ReleaseNotifier {
	return &ReleaseNotifier{
		mailer:           mailer,
		subscriberReader: subscriberReader,
	}
}

func (n *ReleaseNotifier) NotifyConfirmedSubscribers(
	ctx context.Context,
	repo catalog.Repository,
	release *github.Release,
) error {
	subscriptions, err := n.subscriberReader.ListConfirmedByRepositoryID(ctx, repo.ID)
	if err != nil {
		return err
	}

	slog.Info(
		"worker loaded confirmed subscriptions",
		"repository_id", repo.ID,
		"repository", repo.FullName,
		"count", len(subscriptions),
	)

	if len(subscriptions) == 0 {
		return nil
	}

	recipients := make([]notification.ReleaseRecipient, len(subscriptions))
	for i, sub := range subscriptions {
		recipients[i] = notification.ReleaseRecipient{
			Email:            sub.Email,
			UnsubscribeToken: sub.UnsubscribeToken,
		}
	}

	return n.mailer.SendReleaseEmails(recipients, notification.ReleaseInfo{
		Tag:  release.Tag,
		Name: release.Name,
		URL:  release.URL,
	})
}
