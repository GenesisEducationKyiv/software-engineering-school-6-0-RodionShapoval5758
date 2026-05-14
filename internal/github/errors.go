package github

import "errors"

var (
	ErrNotFound           = errors.New("github repository not found")
	ErrRateLimited        = errors.New("github API rate limited")
	ErrUnexpectedResponse = errors.New("unexpected github API response")
)
