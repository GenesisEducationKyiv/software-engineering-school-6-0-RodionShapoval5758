package watcher

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

	logger := slog.Default().With("scan_id", newScanID())

	repositories, err := w.repositoryRepository.ListTracked(scanCtx)
	if err != nil {
		return err
	}

	logger.Info("worker loaded tracked repositories", "count", len(repositories))

	var rateLimited atomic.Bool
	w.processRepositories(scanCtx, cancelScan, &rateLimited, repositories, logger)

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

		go func(r domain.Repository) {
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
	repo domain.Repository,
	cancelScan context.CancelFunc,
	rateLimited *atomic.Bool,
	logger *slog.Logger,
) {
	if errors.Is(err, domain.ErrRateLimited) {
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

func (w *Worker) processRepository(ctx context.Context, repo domain.Repository, logger *slog.Logger) error {
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

	if err := w.repositoryRepository.UpdateLastSeenTag(ctx, repo.ID, release.Tag); err != nil {
		return err
	}

	return w.releaseNotifier.NotifyConfirmedSubscribers(ctx, repo, release)
}

func (w *Worker) getLatestRelease(ctx context.Context, repo domain.Repository, logger *slog.Logger) (*domain.Release, error) {
	release, err := w.githubClient.GetLatestTag(ctx, repo.FullName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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

func newScanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}
