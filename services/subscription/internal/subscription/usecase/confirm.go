package usecase

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
)

type confirmRepository interface {
	Confirm(ctx context.Context, token string) (int64, error)
}

type confirmSagaOrchestrator interface {
	HandleConfirmed(ctx context.Context, subscriptionID int64) error
}

type Confirm struct {
	repo         confirmRepository
	orchestrator confirmSagaOrchestrator
}

func NewConfirm(repo confirmRepository, orchestrator confirmSagaOrchestrator) *Confirm {
	return &Confirm{repo: repo, orchestrator: orchestrator}
}

func (uc *Confirm) Execute(ctx context.Context, token string) error {
	subscriptionID, err := uc.repo.Confirm(ctx, token)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("confirm token not found: %w", subscription.ErrTokenNotFound)
		}
		return fmt.Errorf("confirm subscription: %w", err)
	}

	if err := uc.orchestrator.HandleConfirmed(ctx, subscriptionID); err != nil {
		return fmt.Errorf("advance saga for subscription %d: %w", subscriptionID, err)
	}

	return nil
}
