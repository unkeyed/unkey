// Package auth provides shared bearer token authentication for ctrl ConnectRPC services.
//
// Each service holds its own bearer token and exposes a thin authenticate method
// that delegates to [Authenticate]. This keeps the validation logic in one place
// while letting services decide which handlers require authentication.
package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// Request is satisfied by both *connect.Request and connect.StreamingHandlerConn,
// allowing [Authenticate] to work with unary and streaming handlers.
type Request interface {
	Header() http.Header
}

// Authenticate validates the Authorization header against a preshared bearer token
// using constant-time comparison. Returns connect.CodeUnauthenticated on missing
// header, malformed scheme, or token mismatch.
func Authenticate(req Request, token string) error {
	header := req.Header().Get("Authorization")
	if header == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing Authorization header"))
	}
	if !strings.HasPrefix(header, "Bearer ") {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid Authorization header"))
	}
	bearer := strings.TrimPrefix(header, "Bearer ")

	if subtle.ConstantTimeCompare([]byte(bearer), []byte(token)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid bearer token"))
	}
	return nil
}
