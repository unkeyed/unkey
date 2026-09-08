package middleware

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

// WithReservedHeaderStrip drops internal X-Unkey-* request headers at the
// edge. It preserves signed peer metadata so the proxy handler can verify and
// remove it. It does not preserve metadata or other X-Unkey-* request trailers.
//
// The policy engine relies on internal headers (X-Unkey-Principal especially)
// being trustworthy. Running here protects policy evaluation. Proxy directors
// repeat the trailer check because reading the body can populate trailers after
// this middleware runs.
func WithReservedHeaderStrip() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			req := s.Request()
			for name := range req.Header {
				if proxy.IsUnkeyHeader(name) &&
					name != proxy.HeaderFrontlineMeta {
					delete(req.Header, name)
				}
			}
			for name := range req.Trailer {
				if proxy.IsUnkeyHeader(name) {
					delete(req.Trailer, name)
				}
			}
			return next(ctx, s)
		}
	}
}
