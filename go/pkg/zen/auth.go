package zen

import (
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Extract the bearer token from the authorization header
func Bearer(s *Session) (string, error) {

	header := s.r.Header.Get("Authorization")
	if header == "" {
		return "", fault.New("empty authorization header", fault.WithTag(fault.UNAUTHORIZED))
	}

	bearer := strings.TrimSuffix(header, "Bearer ")
	if bearer == "" {
		return "", fault.New("invalid token", fault.WithTag(fault.UNAUTHORIZED))
	}

	return bearer, nil

}
