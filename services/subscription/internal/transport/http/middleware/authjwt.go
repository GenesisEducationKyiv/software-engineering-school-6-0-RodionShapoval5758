package middleware

import (
	"context"
	"crypto"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type emailContextKey struct{}

type KeyProvider interface {
	Key() (crypto.PublicKey, error)
}

func AuthJWT(keys KeyProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || raw == "" {
				http.Error(w, "Not authorized", http.StatusUnauthorized)

				return
			}

			claims := jwt.MapClaims{}
			_, err := jwt.ParseWithClaims(raw, claims,
				func(_ *jwt.Token) (any, error) { return keys.Key() },
				jwt.WithValidMethods([]string{"ES256"}),
				jwt.WithExpirationRequired(),
			)
			if err != nil {
				http.Error(w, "Not authorized", http.StatusUnauthorized)

				return
			}

			email, _ := claims["email"].(string)
			if email == "" {
				http.Error(w, "Not authorized", http.StatusUnauthorized)

				return
			}

			ctx := context.WithValue(r.Context(), emailContextKey{}, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func EmailFromContext(ctx context.Context) string {
	email, _ := ctx.Value(emailContextKey{}).(string)

	return email
}
