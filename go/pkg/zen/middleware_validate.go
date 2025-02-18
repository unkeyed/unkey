package zen

import (
	"strings"

	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
)

func WithRootKeyAuth(svc keys.KeyService) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(s *Session) error {

			header := s.r.Header.Get("Authorization")
			if header == "" {
				return fault.New("empty authorization header", fault.WithTag(fault.UNAUTHORIZED))
			}

			bearer := strings.TrimSuffix(header, "Bearer ")
			if bearer == "" {
				return fault.New("invalid token", fault.WithTag(fault.UNAUTHORIZED))
			}

			key, err := svc.Verify(s.Context(), hash.Sha256(bearer))
			if err != nil {
				return fault.Wrap(err)
			}

			s.workspaceID = key.AuthorizedWorkspaceID

			return next(s)
		}
	}
}
