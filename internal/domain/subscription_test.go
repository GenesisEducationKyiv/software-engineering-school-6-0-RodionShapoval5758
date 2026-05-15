package domain

import (
	"encoding/base64"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "length 16",
			length:  16,
			wantErr: false,
		},
		{
			name:    "length 32",
			length:  32,
			wantErr: false,
		},
		{
			name:    "length 64",
			length:  64,
			wantErr: false,
		},
		{
			name:    "zero length",
			length:  0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateToken(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateToken() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if tt.wantErr {
				return
			}

			decoded, err := base64.RawURLEncoding.DecodeString(got)
			if err != nil {
				t.Errorf("generateToken() result is not valid base64: %v", err)
			}
			if len(decoded) != tt.length {
				t.Errorf("generateToken() decoded length = %v, want %v", len(decoded), tt.length)
			}

			got2, _ := generateToken(tt.length)
			if tt.length > 0 && got == got2 {
				t.Errorf("generateToken() generated the same token twice: %v", got)
			}
		})
	}
}

func TestNewSubscription(t *testing.T) {
	email := "test@example.com"
	repoID := int64(123)

	sub, err := NewSubscription(email, repoID)
	if err != nil {
		t.Fatalf("NewSubscription() error = %v", err)
	}

	if sub.Email != email {
		t.Errorf("expected email %s, got %s", email, sub.Email)
	}

	if sub.RepositoryID != repoID {
		t.Errorf("expected repoID %d, got %d", repoID, sub.RepositoryID)
	}

	if len(sub.ConfirmToken) == 0 {
		t.Error("ConfirmToken should not be empty")
	}

	if len(sub.UnsubscribeToken) == 0 {
		t.Error("UnsubscribeToken should not be empty")
	}

	if sub.IsConfirmed() {
		t.Error("new subscription should not be confirmed")
	}
}
