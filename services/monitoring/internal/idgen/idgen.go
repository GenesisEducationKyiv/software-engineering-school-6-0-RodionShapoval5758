package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

func New() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}
