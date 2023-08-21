package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
)

// Authorizes the predefined admin key
func (s *Server) authorizeStaticKey(ctx context.Context, header string) error {
	if header == "" {
		return fmt.Errorf("authorization header is empty")
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" {
		return fmt.Errorf("authorization header is malformed")
	}

	if subtle.ConstantTimeCompare([]byte(s.unkeyAppAuthToken), []byte(token)) == 0 {
		return fmt.Errorf("invalid token")
	}
	return nil

}
