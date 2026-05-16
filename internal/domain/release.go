package domain

import "time"

type Release struct {
	Tag         string
	Name        string
	URL         string
	PublishedAt time.Time
}
