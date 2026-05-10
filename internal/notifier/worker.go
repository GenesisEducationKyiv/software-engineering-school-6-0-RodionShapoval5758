package notifier

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/github"
)

type smtpClient interface {
	SendReleaseNotification(toEmail string, unsubscribeToken string, release *github.Release) error
}

type githubClient interface {
	GetLatestTag(ctx context.Context, fullName string) (*github.Release, error)
}

type subscriptionRepository interface {
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.Subscription, error)
}

type repositoryRepository interface {
	ListTracked(ctx context.Context) ([]domain.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repositoryID int64, tag string) error
}

type Worker struct {
	githubClient         githubClient
	repositoryRepository repositoryRepository
	releaseNotifier      *releaseNotifier
}

const maxConcurrentRepositoryScans = 10

func NewWorker(
	smtpClient smtpClient,
	githubClient githubClient,
	subscriptionRepository subscriptionRepository,
	repositoryRepository repositoryRepository,
) *Worker {
	return &Worker{
		githubClient:         githubClient,
		repositoryRepository: repositoryRepository,
		releaseNotifier:      newReleaseNotifier(smtpClient, subscriptionRepository),
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
	scanCtx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()

	repositories, err := w.listTrackedRepositories(scanCtx)
	if err != nil {
		return err
	}

	slog.Info("worker loaded tracked repositories", "count", len(repositories))

	if err := w.processRepositories(scanCtx, cancelScan, repositories); err != nil {
		return err
	}

	if ctx.Err() == nil && scanCtx.Err() != nil {
		return github.ErrRateLimited
	}

	return nil
}

func (w *Worker) listTrackedRepositories(ctx context.Context) ([]domain.Repository, error) {
	return w.repositoryRepository.ListTracked(ctx)
}

func (w *Worker) processRepositories(
	scanCtx context.Context,
	cancelScan context.CancelFunc,
	repositories []domain.Repository,
) error {
	sem := make(chan struct{}, maxConcurrentRepositoryScans)
	var waitGroup sync.WaitGroup
	for _, repo := range repositories {
		if scanCtx.Err() != nil {
			break
		}
		waitGroup.Add(1)
		sem <- struct{}{}
		go func(r domain.Repository) {
			defer waitGroup.Done()
			defer func() { <-sem }()
			if err := w.processRepository(scanCtx, r); err != nil {
				w.handleRepositoryProcessingError(err, r, cancelScan)
			}
		}(repo)
	}
	waitGroup.Wait()

	return nil
}

func (w *Worker) handleRepositoryProcessingError(
	err error,
	repo domain.Repository,
	cancelScan context.CancelFunc,
) {
	if errors.Is(err, github.ErrRateLimited) {
		cancelScan()

		return
	}

	slog.Error(
		"worker repository processing failed",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
		"error",
		err,
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

	if !w.hasNewRelease(repo, release) {
		w.logUnchangedRelease(repo, release)

		return nil
	}

	w.logDetectedNewRelease(repo, release)

	if err := w.markReleaseSeen(ctx, repo.ID, release.Tag); err != nil {
		return err
	}

	return w.notifyConfirmedSubscribers(ctx, repo, release)
}

func (w *Worker) getLatestRelease(ctx context.Context, repo domain.Repository) (*github.Release, error) {
	release, err := w.githubClient.GetLatestTag(ctx, repo.FullName)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			w.logRepositoryWithoutLatestRelease(repo)

			return nil, nil
		}

		return nil, err
	}

	return release, nil
}

func (w *Worker) hasNewRelease(repo domain.Repository, release *github.Release) bool {
	return repo.LastSeenTag == nil || release.Tag != *repo.LastSeenTag
}

func (w *Worker) markReleaseSeen(ctx context.Context, repositoryID int64, tag string) error {
	return w.repositoryRepository.UpdateLastSeenTag(ctx, repositoryID, tag)
}

func (w *Worker) notifyConfirmedSubscribers(ctx context.Context, repo domain.Repository, release *github.Release) error {
	return w.releaseNotifier.notifyConfirmedSubscribers(ctx, repo, release)
}

func (w *Worker) logRepositoryWithoutLatestRelease(repo domain.Repository) {
	slog.Info(
		"worker skipped repository without latest release",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
	)
}

func (w *Worker) logUnchangedRelease(repo domain.Repository, release *github.Release) {
	slog.Info(
		"worker skipped repository with unchanged release tag",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
		"tag",
		release.Tag,
	)
}

func (w *Worker) logDetectedNewRelease(repo domain.Repository, release *github.Release) {
	slog.Info(
		"worker detected new release",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
		"tag",
		release.Tag,
	)
}
