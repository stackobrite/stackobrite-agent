package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

type HMACAuth struct {
	token     []byte
	tokenHash string
}

func New(token string) *HMACAuth {
	return &HMACAuth{
		token:     []byte(token),
		tokenHash: token,
	}
}

func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (a *HMACAuth) Validate(token string) bool {
	return subtle.ConstantTimeCompare([]byte(token), a.token) == 1
}

func (a *HMACAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		if !a.Validate(parts[1]) {
			http.Error(w, `{"error":"invalid token"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
