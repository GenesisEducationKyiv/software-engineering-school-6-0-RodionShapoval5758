package domain

import "errors"

var (
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrInvalidRepoFormat  = errors.New("invalid repository format, has to be owner/repo")
)
