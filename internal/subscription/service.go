package subscription

import (
	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/subscription/internal/domain"
)

type Subscription = domain.Subscription
type SubscriptionDetails = domain.SubscriptionDetails

type Service struct {
	subscriptionRepository subscriptionRepository
	catalogClient          catalogClient
	githubClient           githubClient
	outbox                 outboxWriter
	pool                   db.TxBeginner
}

func NewService(
	repo subscriptionRepository,
	catalog catalogClient,
	github githubClient,
	outbox outboxWriter,
	pool db.TxBeginner,
) *Service {
	return &Service{
		subscriptionRepository: repo,
		catalogClient:          catalog,
		githubClient:           github,
		outbox:                 outbox,
		pool:                   pool,
	}
}
