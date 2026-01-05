package api

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
)

func withTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx, span := tracing.Start(ctx, tracing.NewSpanName("api", r.URL.Path))
		defer span.End()
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
