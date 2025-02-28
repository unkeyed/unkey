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

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "Bearer ") {
		return "", fault.New("invalid format", fault.WithTag(fault.UNAUTHORIZED),
			fault.WithDesc("missing bearer prefix", "Your authorization header is missing the 'Bearer ' prefix."))
	}

	bearer := strings.TrimPrefix(header, "Bearer ")
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return "", fault.New("invalid token", fault.WithTag(fault.UNAUTHORIZED))
	}

	return bearer, nil

}
