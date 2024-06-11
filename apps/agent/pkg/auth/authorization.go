package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
)

func Authorize(ctx context.Context, authorizationHeader string) error {
	if authorizationHeader == "" {
		return ErrUnauthorized
	}

	authToken := os.Getenv("AUTH_TOKEN")

	fmt.Println("authToken:           ", authToken)
	fmt.Println("authorizationHeader: ", authorizationHeader)
	for _, token := range strings.Split(authToken, ",") {
		if subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authorizationHeader, "Bearer ")), []byte(token)) == 1 {
			return nil
		}
	}
	return ErrUnauthorized
}
