package user

import "errors"

var (
	ErrEmailTaken          = errors.New("email is already registered")
	ErrInvalidEmailFormat  = errors.New("invalid email format")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrEmailNotVerified    = errors.New("email is not verified")
	ErrTokenNotFound       = errors.New("verification token not found")
	ErrTokenExpired        = errors.New("verification token expired")
	ErrTokenUsed           = errors.New("verification token already used")
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid, expired, or revoked")
)

type User struct {
	ID            string
	Email         string
	PasswordHash  string
	EmailVerified bool
}
