package service

import (
	"errors"
	"testing"
)

func TestValidateRepoFormat(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr error
	}{
		{
			name:    "valid format",
			repo:    "golang/go",
			wantErr: nil,
		},
		{
			name:    "valid format with spaces",
			repo:    "  owner/repo  ",
			wantErr: nil,
		},
		{
			name:    "missing owner",
			repo:    "/go",
			wantErr: ErrInvalidRepoFormat,
		},
		{
			name:    "missing repo",
			repo:    "golang/",
			wantErr: ErrInvalidRepoFormat,
		},
		{
			name:    "no slash",
			repo:    "golang",
			wantErr: ErrInvalidRepoFormat,
		},
		{
			name:    "too many slashes",
			repo:    "golang/go/src",
			wantErr: ErrInvalidRepoFormat,
		},
		{
			name:    "empty string",
			repo:    "",
			wantErr: ErrInvalidRepoFormat,
		},
		{
			name:    "only slash",
			repo:    "/",
			wantErr: ErrInvalidRepoFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepoFormat(tt.repo)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateRepoFormat(%q) error = %v, wantErr %v", tt.repo, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmailFormat(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with spaces",
			email:   "  test@domain.io  ",
			wantErr: nil,
		},
		{
			name:    "missing @",
			email:   "userexample.com",
			wantErr: ErrInvalidEmailFormat,
		},
		{
			name:    "missing domain",
			email:   "user@",
			wantErr: ErrInvalidEmailFormat,
		},
		{
			name:    "missing user",
			email:   "@example.com",
			wantErr: ErrInvalidEmailFormat,
		},
		{
			name:    "empty string",
			email:   "",
			wantErr: ErrInvalidEmailFormat,
		},
		{
			name:    "invalid characters",
			email:   "user name@example.com",
			wantErr: ErrInvalidEmailFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmailFormat(tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateEmailFormat(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}
