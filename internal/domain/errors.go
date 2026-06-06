package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrTokenConflict      = errors.New("token conflict")
	ErrRateLimited        = errors.New("rate limited")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUnexpectedResponse = errors.New("unexpected github API response")
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrInvalidRepoFormat  = errors.New("invalid repository format, has to be owner/repo")
)
