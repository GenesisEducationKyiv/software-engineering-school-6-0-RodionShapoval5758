package handler

import (
	"context"
	"errors"

	subscriptionv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/subscriptionv1/subscription/v1"
	"GithubReleaseNotificationAPI/services/subscription/api/gen/subscriptionv1/subscription/v1/v1connect"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"

	"connectrpc.com/connect"
)

type subscriber interface {
	Execute(ctx context.Context, email, repo string) error
}

type confirmer interface {
	Execute(ctx context.Context, token string) error
}

type unsubscriber interface {
	Execute(ctx context.Context, token string) error
}

type lister interface {
	ByEmail(ctx context.Context, email string) ([]subscription.SubscriptionDetails, error)
}

type catalogLister interface {
	ListTracked(ctx context.Context) ([]catalog.Repository, error)
}

type SubscriptionHandler struct {
	v1connect.UnimplementedSubscriptionServiceHandler
	subscriber    subscriber
	confirmer     confirmer
	unsubscriber  unsubscriber
	lister        lister
	catalogLister catalogLister
}

func New(sub subscriber, conf confirmer, unsub unsubscriber, list lister, catalog catalogLister) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriber:    sub,
		confirmer:     conf,
		unsubscriber:  unsub,
		lister:        list,
		catalogLister: catalog,
	}
}

func (h *SubscriptionHandler) Subscribe(ctx context.Context, req *connect.Request[subscriptionv1.SubscribeRequest]) (*connect.Response[subscriptionv1.SubscribeResponse], error) {
	if err := h.subscriber.Execute(ctx, req.Msg.Email, req.Msg.Repo); err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&subscriptionv1.SubscribeResponse{
		Message: "Subscription successful. Confirmation email sent",
	}), nil
}

func (h *SubscriptionHandler) Confirm(ctx context.Context, req *connect.Request[subscriptionv1.ConfirmRequest]) (*connect.Response[subscriptionv1.ConfirmResponse], error) {
	if err := h.confirmer.Execute(ctx, req.Msg.Token); err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&subscriptionv1.ConfirmResponse{
		Message: "Subscription confirmed successfully",
	}), nil
}

func (h *SubscriptionHandler) Unsubscribe(ctx context.Context, req *connect.Request[subscriptionv1.UnsubscribeRequest]) (*connect.Response[subscriptionv1.UnsubscribeResponse], error) {
	if err := h.unsubscriber.Execute(ctx, req.Msg.Token); err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&subscriptionv1.UnsubscribeResponse{
		Message: "Unsubscribed successfully",
	}), nil
}

func (h *SubscriptionHandler) ListSubscriptions(ctx context.Context, req *connect.Request[subscriptionv1.ListSubscriptionsRequest]) (*connect.Response[subscriptionv1.ListSubscriptionsResponse], error) {
	details, err := h.lister.ByEmail(ctx, req.Msg.Email)
	if err != nil {
		return nil, toConnectError(err)
	}

	subs := make([]*subscriptionv1.Subscription, 0, len(details))
	for _, d := range details {
		tag := d.LastSeenTag
		if tag == "" {
			tag = "not available yet"
		}

		subs = append(subs, &subscriptionv1.Subscription{
			Email:       d.Email,
			Repo:        d.Repo,
			Confirmed:   d.Confirmed,
			LastSeenTag: tag,
		})
	}

	return connect.NewResponse(&subscriptionv1.ListSubscriptionsResponse{
		Subscriptions: subs,
	}), nil
}

func (h *SubscriptionHandler) ListTrackedRepos(ctx context.Context, _ *connect.Request[subscriptionv1.ListTrackedReposRequest]) (*connect.Response[subscriptionv1.ListTrackedReposResponse], error) {
	repos, err := h.catalogLister.ListTracked(ctx)
	if err != nil {
		return nil, toConnectError(err)
	}

	out := make([]*subscriptionv1.TrackedRepo, 0, len(repos))
	for _, r := range repos {
		out = append(out, &subscriptionv1.TrackedRepo{
			RepoId:   r.ID,
			FullName: r.FullName,
		})
	}

	return connect.NewResponse(&subscriptionv1.ListTrackedReposResponse{Repos: out}), nil
}

func toConnectError(err error) error {
	switch {
	case errors.Is(err, subscription.ErrInvalidEmailFormat):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, subscription.ErrInvalidRepoFormat):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, subscription.ErrTokenNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, subscription.ErrRepoNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, subscription.ErrSubscriptionAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, subscription.ErrTooMuchRequests):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, subscription.ErrGitHubUnauthorized):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
