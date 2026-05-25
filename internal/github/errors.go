package github

import "errors"

var (
	ErrNotFound           = errors.New("github repository not found")
	ErrRateLimited        = errors.New("github API rate limited")
	ErrUnauthorized       = errors.New("github API unauthorized: token is invalid or expired")
	ErrUnexpectedResponse = errors.New("unexpected github API response")
)
