package handler

import (
	"context"

	catalogv1 "GithubReleaseNotificationAPI/services/subscription/api/gen/catalogv1/catalog/v1"
	"GithubReleaseNotificationAPI/services/subscription/internal/catalog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type catalogLister interface {
	ListTracked(ctx context.Context) ([]catalog.Repository, error)
}

type CatalogHandler struct {
	catalogv1.UnimplementedCatalogServiceServer
	lister catalogLister
}

func NewCatalog(lister catalogLister) *CatalogHandler {
	return &CatalogHandler{lister: lister}
}

func (h *CatalogHandler) ListTrackedRepos(ctx context.Context, _ *catalogv1.ListTrackedReposRequest) (*catalogv1.ListTrackedReposResponse, error) {
	repos, err := h.lister.ListTracked(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	out := make([]*catalogv1.TrackedRepo, 0, len(repos))
	for _, r := range repos {
		out = append(out, &catalogv1.TrackedRepo{
			RepoId:   r.ID,
			FullName: r.FullName,
		})
	}

	return &catalogv1.ListTrackedReposResponse{Repos: out}, nil
}
