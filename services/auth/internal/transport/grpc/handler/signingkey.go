package handler

import (
	"context"

	authv1 "GithubReleaseNotificationAPI/services/auth/api/gen/authv1/auth/v1"
	"GithubReleaseNotificationAPI/services/auth/internal/token"
)

type keySource interface {
	KID() string
	PublicKeyPEM() []byte
}

type SigningKey struct {
	authv1.UnimplementedAuthServiceServer

	keys keySource
}

func NewSigningKey(keys keySource) *SigningKey {
	return &SigningKey{keys: keys}
}

func (s *SigningKey) GetSigningKey(_ context.Context, _ *authv1.GetSigningKeyRequest) (*authv1.GetSigningKeyResponse, error) {
	return &authv1.GetSigningKeyResponse{
		Kid:          s.keys.KID(),
		PublicKeyPem: s.keys.PublicKeyPEM(),
		Algorithm:    token.Algorithm,
	}, nil
}
