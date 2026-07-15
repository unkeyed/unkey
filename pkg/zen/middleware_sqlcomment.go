package zen

import (
	"context"

	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

// WithSQLComment attaches the matched HTTP route pattern to the request context
// so downstream MySQL queries include a low-cardinality route tag in PlanetScale
// Insights. Uses [http.Request.Pattern] from the Go 1.22 ServeMux (for example
// "POST /v2/keys.verifyKey").
func WithSQLComment() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			if pattern := s.Request().Pattern; pattern != "" {
				ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{
					Route:  pattern,
					Source: "http",
				})
			}
			return next(ctx, s)
		}
	}
}
