package notifier

import (
	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/github"
	"context"
	"log/slog"
)

type releaseNotifier struct {
	smtpClient             smtpClient
	subscriptionRepository subscriptionRepository
}

func newReleaseNotifier(
	smtpClient smtpClient,
	subscriptionRepository subscriptionRepository,
) *releaseNotifier {
	return &releaseNotifier{
		smtpClient:             smtpClient,
		subscriptionRepository: subscriptionRepository,
	}
}

func (n *releaseNotifier) notifyConfirmedSubscribers(
	ctx context.Context,
	repo domain.Repository,
	release *github.Release,
) error {
	subscriptions, err := n.subscriptionRepository.ListConfirmedByRepositoryID(ctx, repo.ID)
	if err != nil {
		return err
	}

	slog.Info(
		"worker loaded confirmed subscriptions",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
		"count",
		len(subscriptions),
	)

	for _, subscription := range subscriptions {
		n.sendReleaseNotification(repo, subscription, release)
	}

	return nil
}

func (n *releaseNotifier) sendReleaseNotification(
	repo domain.Repository,
	subscription domain.Subscription,
	release *github.Release,
) {
	err := n.smtpClient.SendReleaseNotification(subscription.Email, subscription.UnsubscribeToken, release)
	if err != nil {
		slog.Error(
			"worker notification send failed",
			"repository_id",
			repo.ID,
			"repository",
			repo.FullName,
			"email",
			subscription.Email,
			"error",
			err,
		)

		return
	}

	slog.Info(
		"worker notification sent",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
		"email",
		subscription.Email,
		"tag",
		release.Tag,
	)
}
