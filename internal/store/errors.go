package store

import "errors"

var (
	ErrNotFound            = errors.New("row not found")
	ErrAlreadyExists       = errors.New("row already exists")
	ErrTokensAlreadyExists = errors.New("token exists")
)
