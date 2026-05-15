package domain

import "time"

type Repository struct {
	ID          int64
	FullName    string
	LastSeenTag *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *Repository) HasNewRelease(tag string) bool {
	return r.LastSeenTag == nil || tag != *r.LastSeenTag
}
