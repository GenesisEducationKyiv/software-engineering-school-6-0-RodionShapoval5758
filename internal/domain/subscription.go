package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

const TokenLength = 32

type Subscription struct {
	ID               int64
	Email            string
	RepositoryID     int64
	Confirmed        bool
	ConfirmToken     string
	UnsubscribeToken string
	CreatedAt        time.Time
	ConfirmedAt      *time.Time
}

func NewSubscription(email string, repositoryID int64) (*Subscription, error) {
	confirmToken, err := generateToken(TokenLength)
	if err != nil {
		return nil, fmt.Errorf("generate confirm token: %w", err)
	}

	unsubscribeToken, err := generateToken(TokenLength)
	if err != nil {
		return nil, fmt.Errorf("generate unsubscribe token: %w", err)
	}

	return &Subscription{
		Email:            email,
		RepositoryID:     repositoryID,
		ConfirmToken:     confirmToken,
		UnsubscribeToken: unsubscribeToken,
	}, nil
}

func (s *Subscription) IsConfirmed() bool {
	return s.Confirmed
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
