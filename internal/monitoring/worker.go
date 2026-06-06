package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"GithubReleaseNotificationAPI/internal/catalog"
	"GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/idgen"
	"GithubReleaseNotificationAPI/internal/shared"
)

type releaseNotifier interface {
	NotifyConfirmedSubscribers(ctx context.Context, repo catalog.Repository, release *github.Release) error
}

type Worker struct {
	githubClient    githubClient
	catalogClient   catalogClient
	releaseNotifier releaseNotifier
	metrics         scanObserver
}

const maxConcurrentRepositoryScans = 10

func NewWorker(
	githubClient githubClient,
	catalogClient catalogClient,
	releaseNotifier releaseNotifier,
	metrics scanObserver,
) *Worker {
	return &Worker{
		githubClient:    githubClient,
		catalogClient:   catalogClient,
		releaseNotifier: releaseNotifier,
		metrics:         metrics,
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
	if errors.Is(err, github.ErrRateLimited) {
		slog.Warn("GitHub API rate limit exceeded. Pausing scanner until next interval.")

		return
	}

	slog.Error("worker scan pass failed unexpectedly", "error", err)
}

func (w *Worker) runOneScan(ctx context.Context) error {
	start := time.Now()

	scanCtx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()

	logger := slog.Default().With("scan_id", idgen.New())

	repositories, err := w.catalogClient.ListTracked(scanCtx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		w.observeScan(time.Since(start), "error")
		return err
	}

	logger.Info("worker loaded tracked repositories", "count", len(repositories))

	var rateLimited atomic.Bool
	w.processRepositories(scanCtx, cancelScan, &rateLimited, repositories, logger)

	if rateLimited.Load() {
		w.observeScan(time.Since(start), "rate_limited")
		return github.ErrRateLimited
	}

	w.observeScan(time.Since(start), "ok")

	return nil
}

func (w *Worker) observeScan(d time.Duration, result string) {
	if w.metrics == nil {
		return
	}
	w.metrics.ObserveScanDuration(d.Seconds())
	w.metrics.IncScanResult(result)
}

func (w *Worker) processRepositories(
	scanCtx context.Context,
	cancelScan context.CancelFunc,
	rateLimited *atomic.Bool,
	repositories []catalog.Repository,
	logger *slog.Logger,
) {
	sem := make(chan struct{}, maxConcurrentRepositoryScans)
	var wg sync.WaitGroup

	for _, repo := range repositories {
		if scanCtx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(r catalog.Repository) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := w.processRepository(scanCtx, r, logger); err != nil {
				w.handleRepositoryProcessingError(err, r, cancelScan, rateLimited, logger)
			}
		}(repo)
	}

	wg.Wait()
}

func (w *Worker) handleRepositoryProcessingError(
	err error,
	repo catalog.Repository,
	cancelScan context.CancelFunc,
	rateLimited *atomic.Bool,
	logger *slog.Logger,
) {
	if errors.Is(err, github.ErrRateLimited) {
		rateLimited.Store(true)
		cancelScan()

		return
	}

	logger.Error(
		"worker repository processing failed",
		"repository_id", repo.ID,
		"repository", repo.FullName,
		"error", err,
	)
}

func (w *Worker) processRepository(ctx context.Context, repo catalog.Repository, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	release, err := w.getLatestRelease(ctx, repo, logger)
	if err != nil {
		return err
	}

	if release == nil {
		return nil
	}

	if !repo.HasNewRelease(release.Tag) {
		logger.Info(
			"worker skipped repository with unchanged release tag",
			"repository_id", repo.ID,
			"repository", repo.FullName,
			"tag", release.Tag,
		)

		return nil
	}

	logger.Info(
		"worker detected new release",
		"repository_id", repo.ID,
		"repository", repo.FullName,
		"tag", release.Tag,
	)

	if err := w.catalogClient.UpdateLastSeenTag(ctx, repo.ID, release.Tag); err != nil {
		return err
	}

	return w.releaseNotifier.NotifyConfirmedSubscribers(ctx, repo, release)
}

func (w *Worker) getLatestRelease(ctx context.Context, repo catalog.Repository, logger *slog.Logger) (*github.Release, error) {
	release, err := w.githubClient.GetLatestTag(ctx, repo.FullName)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			logger.Info(
				"worker skipped repository without latest release",
				"repository_id", repo.ID,
				"repository", repo.FullName,
			)

			return nil, nil
		}

		return nil, err
	}

	return release, nil
}
