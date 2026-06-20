package usecase

import (
	"context"
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/internal/db"
	"GithubReleaseNotificationAPI/internal/subscription"
)

type confirmRepository interface {
	Confirm(ctx context.Context, token string) error
}

type Confirm struct {
	repo confirmRepository
}

func NewConfirm(repo confirmRepository) *Confirm {
	return &Confirm{repo: repo}
}

func (uc *Confirm) Execute(ctx context.Context, token string) error {
	if err := uc.repo.Confirm(ctx, token); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("confirm token not found: %w", subscription.ErrTokenNotFound)
		}

		return fmt.Errorf("confirm subscription: %w", err)
	}

	return nil
}
