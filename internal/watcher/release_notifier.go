package watcher

import (
	"context"
	"log/slog"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/notification"
)

type smtpClient interface {
	SendReleaseEmails(recipients []notification.ReleaseRecipient, release notification.ReleaseInfo) error
}

type subscriptionRepository interface {
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error)
}

type ReleaseNotifier struct {
	smtpClient             smtpClient
	subscriptionRepository subscriptionRepository
}

func NewReleaseNotifier(
	smtpClient smtpClient,
	subscriptionRepository subscriptionRepository,
) *ReleaseNotifier {
	return &ReleaseNotifier{
		smtpClient:             smtpClient,
		subscriptionRepository: subscriptionRepository,
	}
}

func (n *ReleaseNotifier) NotifyConfirmedSubscribers(
	ctx context.Context,
	repo domain.Repository,
	release *domain.Release,
) error {
	subscriptions, err := n.subscriptionRepository.ListConfirmedByRepositoryID(ctx, repo.ID)
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

	return n.smtpClient.SendReleaseEmails(recipients, notification.ReleaseInfo{
		Tag:  release.Tag,
		Name: release.Name,
		URL:  release.URL,
	})
}
