package domain

import "time"

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
