package notifier

import (
	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/github"
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
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
	smtpClient             smtpClient
	githubClient           githubClient
	subscriptionRepository subscriptionRepository
	repositoryRepository   repositoryRepository
}

func NewWorker(
	smtpClient smtpClient,
	githubClient githubClient,
	subscriptionRepository subscriptionRepository,
	repositoryRepository repositoryRepository,
) *Worker {
	return &Worker{
		smtpClient:             smtpClient,
		githubClient:           githubClient,
		subscriptionRepository: subscriptionRepository,
		repositoryRepository:   repositoryRepository,
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

	repositories, err := w.repositoryRepository.ListTracked(scanCtx)
	if err != nil {
		return err
	}

	slog.Info("worker loaded tracked repositories", "count", len(repositories))

	sem := make(chan struct{}, 10)
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
				if errors.Is(err, github.ErrRateLimited) {
					cancelScan()
					return
				}
				slog.Error(
					"worker repository processing failed",
					"repository_id",
					r.ID,
					"repository",
					r.FullName,
					"error",
					err,
				)
				return
			}
		}(repo)
	}
	waitGroup.Wait()

	if ctx.Err() == nil && scanCtx.Err() != nil {
		return github.ErrRateLimited
	}

	return nil
}

func (w *Worker) processRepository(ctx context.Context, repo domain.Repository) error {
	release, err := w.githubClient.GetLatestTag(ctx, repo.FullName)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			slog.Info(
				"worker skipped repository without latest release",
				"repository_id",
				repo.ID,
				"repository",
				repo.FullName,
			)
			return nil
		}
		return err
	}

	if repo.LastSeenTag != nil && release.Tag == *repo.LastSeenTag {
		slog.Info(
			"worker skipped repository with unchanged release tag",
			"repository_id",
			repo.ID,
			"repository",
			repo.FullName,
			"tag",
			release.Tag,
		)
		return nil
	}

	slog.Info(
		"worker detected new release",
		"repository_id",
		repo.ID,
		"repository",
		repo.FullName,
		"tag",
		release.Tag,
	)

	if err := w.repositoryRepository.UpdateLastSeenTag(ctx, repo.ID, release.Tag); err != nil {
		return err
	}

	subscriptions, err := w.subscriptionRepository.ListConfirmedByRepositoryID(ctx, repo.ID)
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

	for _, subscription := range subscriptions {
		err := w.smtpClient.SendReleaseNotification(subscription.Email, subscription.UnsubscribeToken, release)
		if err != nil {
			slog.Error(
				"worker notification send failed",
				"repository_id",
				repo.ID,
				"repository",
				repo.FullName,
				"email",
				subscription.Email,
				"error",
				err,
			)
			continue
		}

		slog.Info(
			"worker notification sent",
			"repository_id",
			repo.ID,
			"repository",
			repo.FullName,
			"email",
			subscription.Email,
			"tag",
			release.Tag,
		)
	}

	return nil
}
