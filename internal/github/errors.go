package github

import "errors"

var (
	ErrRateLimited        = errors.New("rate limited")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUnexpectedResponse = errors.New("unexpected github API response")
)
