package httpclient

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

// New returns a reusable standard HTTP client for one downstream service.
// Use the downstream hostname, such as api.resend.com, as its metric label.
func New(hostname string, options ...Option) *http.Client {
	c := config{
		timeout:  30 * time.Second,
		attempts: 3,
		backoff: func(n int) time.Duration {
			limit := min(5*time.Second, 100*time.Millisecond<<min(n-1, 6))
			return time.Duration(rand.Int64N(int64(limit)))
		},
		transport: http.DefaultTransport,
		breaker: circuitbreaker.New[*http.Response](hostname, circuitbreaker.WithIsDownstreamError(func(err error) bool {
			return err != nil && !errors.Is(err, context.Canceled)
		})),
		tracing: []otelhttp.Option{otelhttp.WithPropagators(propagation.TraceContext{})},
		headers: make(http.Header),
	}
	for _, option := range options {
		option(&c)
	}
	return &http.Client{Timeout: c.timeout, Transport: &transport{
		config: c,
		next:   otelhttp.NewTransport(c.transport, c.tracing...),
	}}
}
