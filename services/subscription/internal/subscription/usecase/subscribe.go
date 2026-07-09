package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"GithubReleaseNotificationAPI/contract"
	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	githubclient "GithubReleaseNotificationAPI/services/subscription/internal/github"
	"GithubReleaseNotificationAPI/services/subscription/internal/idgen"
	"GithubReleaseNotificationAPI/services/subscription/internal/saga"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription/internal/domain"

	"github.com/jackc/pgx/v5"
)

type subscribeRepository interface {
	CreateInTx(ctx context.Context, q db.DBTX, sub domain.Subscription) (int64, error)
}

type subscribeCatalog interface {
	Ensure(ctx context.Context, fullName string) (int64, error)
}

type subscribeGithub interface {
	CheckRepo(ctx context.Context, fullName string) error
}

type subscribeOutbox interface {
	Insert(ctx context.Context, q db.DBTX, subject string, payload []byte) error
}

type subscribeSaga interface {
	InsertInTx(ctx context.Context, q db.DBTX, r saga.Row) error
}

const maxTokenAttempts = 5

type Subscribe struct {
	repo      subscribeRepository
	catalog   subscribeCatalog
	github    subscribeGithub
	outbox    subscribeOutbox
	sagaStore subscribeSaga
	pool      db.TxBeginner
	sagaTTL   time.Duration
}

func NewSubscribe(
	repo subscribeRepository,
	catalog subscribeCatalog,
	github subscribeGithub,
	outbox subscribeOutbox,
	sagaStore subscribeSaga,
	pool db.TxBeginner,
	sagaTTL time.Duration,
) *Subscribe {
	return &Subscribe{
		repo:      repo,
		catalog:   catalog,
		github:    github,
		outbox:    outbox,
		sagaStore: sagaStore,
		pool:      pool,
		sagaTTL:   sagaTTL,
	}
}

func (uc *Subscribe) Execute(ctx context.Context, email, repo string) error {
	email, repo, err := normalizeSubscribeInput(email, repo)
	if err != nil {
		return err
	}

	if err := uc.verifyRepo(ctx, repo); err != nil {
		return err
	}

	repositoryID, err := uc.catalog.Ensure(ctx, repo)
	if err != nil {
		return fmt.Errorf("ensure repository %s: %w", repo, err)
	}

	return uc.createPending(ctx, email, repo, repositoryID)
}

func normalizeSubscribeInput(email, repo string) (string, string, error) {
	email = strings.TrimSpace(email)
	if err := domain.ValidateEmail(email); err != nil {
		return "", "", err
	}

	repo = strings.TrimSpace(repo)
	if err := domain.ValidateRepo(repo); err != nil {
		return "", "", subscription.ErrInvalidRepoFormat
	}

	return email, repo, nil
}

func (uc *Subscribe) verifyRepo(ctx context.Context, repo string) error {
	if err := uc.github.CheckRepo(ctx, repo); err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			return subscription.ErrRepoNotFound
		case errors.Is(err, githubclient.ErrRateLimited):
			return subscription.ErrTooMuchRequests
		case errors.Is(err, githubclient.ErrUnauthorized):
			return subscription.ErrGitHubUnauthorized
		default:
			return fmt.Errorf("github repo check failed: %w", err)
		}
	}

	return nil
}

func (uc *Subscribe) createPending(ctx context.Context, email, repoName string, repositoryID int64) error {
	sagaID := idgen.New()

	for range maxTokenAttempts {
		sub, err := domain.NewSubscription(email, repositoryID)
		if err != nil {
			return fmt.Errorf("prepare domain subscription: %w", err)
		}

		confirmPayload, err := json.Marshal(contract.ConfirmationRequested{
			SagaID:       sagaID,
			Email:        email,
			RepoName:     repoName,
			ConfirmToken: sub.ConfirmToken,
		})
		if err != nil {
			return fmt.Errorf("marshal confirmation event: %w", err)
		}

		err = uc.createAndEnqueue(ctx, sagaID, *sub, confirmPayload)
		if errors.Is(err, db.ErrTokenConflict) {
			continue
		}

		if err != nil {
			if errors.Is(err, db.ErrAlreadyExists) {
				return subscription.ErrSubscriptionAlreadyExists
			}

			return fmt.Errorf("failed to create subscription: %w", err)
		}

		return nil
	}

	return fmt.Errorf("create subscription tokens conflict after retries: %w", db.ErrTokenConflict)
}

func (uc *Subscribe) createAndEnqueue(ctx context.Context, sagaID string, sub domain.Subscription, confirmPayload []byte) error {
	tx, err := uc.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	subscriptionID, err := uc.repo.CreateInTx(ctx, tx, sub)
	if err != nil {
		return err
	}

	sagaRow := saga.Row{
		SagaID:         sagaID,
		SubscriptionID: subscriptionID,
		RepositoryID:   sub.RepositoryID,
		Email:          sub.Email,
		State:          saga.StateStarted,
		DeadlineAt:     time.Now().Add(uc.sagaTTL),
	}

	if err := uc.sagaStore.InsertInTx(ctx, tx, sagaRow); err != nil {
		return fmt.Errorf("insert saga: %w", err)
	}

	if err := uc.outbox.Insert(ctx, tx, contract.SubjectConfirmation, confirmPayload); err != nil {
		return fmt.Errorf("enqueue confirmation event: %w", err)
	}

	return tx.Commit(ctx)
}
