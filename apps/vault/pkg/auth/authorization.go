package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"os"
	"strings"
)



var (
	
	ErrUnauthorized = errors.New("unauthorized")

)

func Authorize(ctx context.Context, authorizationHeader string ) error {
	if authorizationHeader == "" {
		return ErrUnauthorized
	}

	secret := os.Getenv("AUTH_SECRET")
	if subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authorizationHeader, "Bearer ")), []byte(secret)) != 1 {
		return ErrUnauthorized
	}
	return nil
}