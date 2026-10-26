package vault

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

type request interface {
	Header() http.Header
}

func (s *Service) authenticate(ctx context.Context, req request) error {
	header := req.Header().Get("Authorization")
	if header == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing Authorization header"))
	}
	if !strings.HasPrefix(header, "Bearer ") {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid Authorization header"))
	}
	bearer := strings.TrimPrefix(header, "Bearer ")

	if s.oidc != nil {
		if err := s.oidc.Verify(ctx, bearer); err != nil {
			return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid workload token"))
		}
		return nil
	}
	if bearer == "" || subtle.ConstantTimeCompare([]byte(bearer), []byte(s.bearer)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid bearer token"))
	}
	return nil
}
