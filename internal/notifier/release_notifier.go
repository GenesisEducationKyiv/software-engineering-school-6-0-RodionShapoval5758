package notifier

import (
	"GithubReleaseNotificationAPI/internal/domain"
	"context"
	"log/slog"
)

type smtpClient interface {
	SendReleaseNotification(toEmail string, unsubscribeToken string, release *domain.Release) error
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

func (n *ReleaseNotifier) sendReleaseNotification(
	repo domain.Repository,
	subscription domain.Subscription,
	release *domain.Release,
) {
	err := n.smtpClient.SendReleaseNotification(subscription.Email, subscription.UnsubscribeToken, release)
	if err != nil {
		slog.Error(
			"worker notification send failed",
			"repository_id",
			repo.ID,
			"repository",
			repo.FullName,
			"subscription_id",
			subscription.ID,
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
		"subscription_id",
		subscription.ID,
		"tag",
		release.Tag,
	)
}
