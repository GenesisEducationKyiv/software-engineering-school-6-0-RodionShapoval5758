package service

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
			got, err := GenerateToken(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			decoded, err := base64.RawURLEncoding.DecodeString(got)
			if err != nil {
				t.Errorf("GenerateToken() result is not valid base64: %v", err)
			}
			if len(decoded) != tt.length {
				t.Errorf("GenerateToken() decoded length = %v, want %v", len(decoded), tt.length)
			}

			got2, _ := GenerateToken(tt.length)
			if tt.length > 0 && got == got2 {
				t.Errorf("GenerateToken() generated the same token twice: %v", got)
			}
		})
	}
}
