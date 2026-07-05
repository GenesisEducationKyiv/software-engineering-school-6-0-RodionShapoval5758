package token

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Algorithm = "ES256"

type Signer struct {
	key       *ecdsa.PrivateKey
	kid       string
	publicPEM []byte
	ttl       time.Duration
}

func LoadSigner(privateKeyPath string, ttl time.Duration) (*Signer, error) {
	raw, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read jwt private key: %w", err)
	}

	key, err := jwt.ParseECPrivateKeyFromPEM(raw)
	if err != nil {
		return nil, fmt.Errorf("parse jwt private key: %w", err)
	}

	spki, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal jwt public key: %w", err)
	}

	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: spki})
	if publicPEM == nil {
		return nil, errors.New("encode jwt public key PEM")
	}

	sum := sha256.Sum256(spki)

	return &Signer{
		key:       key,
		kid:       hex.EncodeToString(sum[:8]),
		publicPEM: publicPEM,
		ttl:       ttl,
	}, nil
}

func (s *Signer) Issue(userID, email string) (string, error) {
	now := time.Now()

	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(s.ttl).Unix(),
	})
	tok.Header["kid"] = s.kid

	signed, err := tok.SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signed, nil
}

func (s *Signer) KID() string {
	return s.kid
}

func (s *Signer) PublicKeyPEM() []byte {
	return s.publicPEM
}
