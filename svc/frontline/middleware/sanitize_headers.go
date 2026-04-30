package middleware

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/pkg/zen"
)

// reservedHeaderPrefix matches headers frontline produces internally and must
// never accept from a client. The prefix is in canonical form (matching the
// keys [http.Header] stores after [http.CanonicalHeaderKey]) so direct prefix
// matching against map keys is correct.
const reservedHeaderPrefix = "X-Unkey-"

// WithReservedHeaderStrip drops every X-Unkey-* request header at the edge.
// This is the single guaranteed sanitization point: the policy engine and the
// upstream both rely on these headers (X-Unkey-Principal especially) being
// trustworthy. Running here, before any routing or policy logic, makes it
// impossible for a code path to forget the strip.
func WithReservedHeaderStrip() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			h := s.Request().Header
			for name := range h {
				if strings.HasPrefix(name, reservedHeaderPrefix) {
					delete(h, name)
				}
			}
			return next(ctx, s)
		}
	}
}
