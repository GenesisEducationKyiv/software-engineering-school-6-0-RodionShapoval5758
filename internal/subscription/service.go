package subscription

import "GithubReleaseNotificationAPI/internal/subscription/internal/domain"

type Subscription = domain.Subscription
type SubscriptionDetails = domain.SubscriptionDetails

type Service struct {
	subscriptionRepository subscriptionRepository
	catalogClient          catalogClient
	githubClient           githubClient
	notifier               notifier
}

func NewService(
	repo subscriptionRepository,
	catalog catalogClient,
	github githubClient,
	notifier notifier,
) *Service {
	return &Service{
		subscriptionRepository: repo,
		catalogClient:          catalog,
		githubClient:           github,
		notifier:               notifier,
	}
}
