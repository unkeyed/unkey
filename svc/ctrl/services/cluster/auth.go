package cluster

import (
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// authenticate validates the request's bearer token against the service's preshared token.
func (s *Service) authenticate(req auth.Request) error {
	return auth.Authenticate(req, s.bearer)
}
