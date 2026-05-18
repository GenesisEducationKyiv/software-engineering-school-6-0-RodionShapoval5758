package notifier

import (
	"context"
	"log/slog"

	"GithubReleaseNotificationAPI/internal/domain"
)

type smtpClient interface {
	SendReleaseNotifications(subscriptions []domain.Subscription, release *domain.Release) error
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

	return n.smtpClient.SendReleaseNotifications(subscriptions, release)
}
