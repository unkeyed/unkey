package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"os"
	"strings"
)

var (
	ErrMissingBearerToken = errors.New("missing bearer token")
	ErrUnauthorized       = errors.New("unauthorized")
)

func Authorize(ctx context.Context, authorizationHeader string) error {
	if authorizationHeader == "" {
		return ErrMissingBearerToken
	}

	authToken := os.Getenv("AUTH_TOKEN")

	for _, token := range strings.Split(authToken, ",") {
		if subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authorizationHeader, "Bearer ")), []byte(token)) == 1 {
			return nil
		}
	}
	return ErrUnauthorized
}
