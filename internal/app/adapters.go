package app

import (
	"context"

	"GithubReleaseNotificationAPI/internal/monitoring"
	"GithubReleaseNotificationAPI/internal/subscription"
)

type confirmedSubProvider interface {
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]subscription.Subscription, error)
}

type ConfirmedSubReader struct {
	svc confirmedSubProvider
}

func NewConfirmedSubReader(svc confirmedSubProvider) *ConfirmedSubReader {
	return &ConfirmedSubReader{svc: svc}
}

func (r *ConfirmedSubReader) ListConfirmedByRepositoryID(ctx context.Context, id int64) ([]monitoring.ConfirmedSubscriber, error) {
	subs, err := r.svc.ListConfirmedByRepositoryID(ctx, id)
	if err != nil {
		return nil, err
	}

	cs := make([]monitoring.ConfirmedSubscriber, len(subs))
	for i, s := range subs {
		cs[i] = monitoring.ConfirmedSubscriber{
			Email:            s.Email,
			UnsubscribeToken: s.UnsubscribeToken,
		}
	}

	return cs, nil
}
