package cluster

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

type request interface {
	Header() http.Header
}

// authenticate validates the bearer token from the request's Authorization header.
// Returns a connect.CodeUnauthenticated error if the token is missing, malformed, or invalid.
func (s *Service) authenticate(req request) error {

	header := req.Header().Get("Authorization")
	if header == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing Authorization header"))
	}
	if !strings.HasPrefix(header, "Bearer ") {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid Authorization header"))
	}
	bearer := strings.TrimPrefix(header, "Bearer ")

	if subtle.ConstantTimeCompare([]byte(bearer), []byte(s.bearer)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid bearer token"))
	}
	return nil

}
