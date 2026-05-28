package watcher

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"GithubReleaseNotificationAPI/internal/domain"
)

type githubClient interface {
	GetLatestTag(ctx context.Context, fullName string) (*domain.Release, error)
}

type repositoryRepository interface {
	ListTracked(ctx context.Context) ([]domain.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repositoryID int64, tag string) error
}

type releaseNotifier interface {
	NotifyConfirmedSubscribers(ctx context.Context, repo domain.Repository, release *domain.Release) error
}

type Worker struct {
	githubClient         githubClient
	repositoryRepository repositoryRepository
	releaseNotifier      releaseNotifier
}

const maxConcurrentRepositoryScans = 10

func NewWorker(
	githubClient githubClient,
	repositoryRepository repositoryRepository,
	releaseNotifier releaseNotifier,
) *Worker {
	return &Worker{
		githubClient:         githubClient,
		repositoryRepository: repositoryRepository,
		releaseNotifier:      releaseNotifier,
	}
}

func (w *Worker) Start(ctx context.Context, loopDuration time.Duration) error {
	slog.Info("worker initial scan started")
	if err := w.runOneScan(ctx); err != nil {
		w.handleScanError(err)
	}

	ticker := time.NewTicker(loopDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			slog.Info("worker scheduled scan started")
			if err := w.runOneScan(ctx); err != nil {
				w.handleScanError(err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Worker) handleScanError(err error) {
	if errors.Is(err, domain.ErrRateLimited) {
		slog.Warn("GitHub API rate limit exceeded. Pausing scanner until next interval.")

		return
	}

	slog.Error("worker scan pass failed unexpectedly", "error", err)
}

func (w *Worker) runOneScan(ctx context.Context) error {
	scanCtx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()

	repositories, err := w.repositoryRepository.ListTracked(scanCtx)
	if err != nil {
		return err
	}

	slog.Info("worker loaded tracked repositories", "count", len(repositories))

	var rateLimited atomic.Bool
	w.processRepositories(scanCtx, cancelScan, &rateLimited, repositories)

	if rateLimited.Load() {
		return domain.ErrRateLimited
	}

	return nil
}

func (w *Worker) processRepositories(
	scanCtx context.Context,
	cancelScan context.CancelFunc,
	rateLimited *atomic.Bool,
	repositories []domain.Repository,
) {
	sem := make(chan struct{}, maxConcurrentRepositoryScans)
	var wg sync.WaitGroup

	for _, repo := range repositories {
		if scanCtx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(r domain.Repository) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := w.processRepository(scanCtx, r); err != nil {
				w.handleRepositoryProcessingError(err, r, cancelScan, rateLimited)
			}
		}(repo)
	}

	wg.Wait()
}

func (w *Worker) handleRepositoryProcessingError(
	err error,
	repo domain.Repository,
	cancelScan context.CancelFunc,
	rateLimited *atomic.Bool,
) {
	if errors.Is(err, domain.ErrRateLimited) {
		rateLimited.Store(true)
		cancelScan()

		return
	}

	slog.Error(
		"worker repository processing failed",
		"repository_id", repo.ID,
		"repository", repo.FullName,
		"error", err,
	)
}

func (w *Worker) processRepository(ctx context.Context, repo domain.Repository) error {
	release, err := w.getLatestRelease(ctx, repo)
	if err != nil {
		return err
	}

	if release == nil {
		return nil
	}

	if !repo.HasNewRelease(release.Tag) {
		w.logUnchangedRelease(repo, release)

		return nil
	}

	w.logDetectedNewRelease(repo, release)

	if err := w.repositoryRepository.UpdateLastSeenTag(ctx, repo.ID, release.Tag); err != nil {
		return err
	}

	return w.releaseNotifier.NotifyConfirmedSubscribers(ctx, repo, release)
}

func (w *Worker) getLatestRelease(ctx context.Context, repo domain.Repository) (*domain.Release, error) {
	release, err := w.githubClient.GetLatestTag(ctx, repo.FullName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			w.logRepositoryWithoutLatestRelease(repo)

			return nil, nil
		}

		return nil, err
	}

	return release, nil
}

func (w *Worker) logRepositoryWithoutLatestRelease(repo domain.Repository) {
	slog.Info(
		"worker skipped repository without latest release",
		"repository_id", repo.ID,
		"repository", repo.FullName,
	)
}

func (w *Worker) logUnchangedRelease(repo domain.Repository, release *domain.Release) {
	slog.Info(
		"worker skipped repository with unchanged release tag",
		"repository_id", repo.ID,
		"repository", repo.FullName,
		"tag", release.Tag,
	)
}

func (w *Worker) logDetectedNewRelease(repo domain.Repository, release *domain.Release) {
	slog.Info(
		"worker detected new release",
		"repository_id", repo.ID,
		"repository", repo.FullName,
		"tag", release.Tag,
	)
}
