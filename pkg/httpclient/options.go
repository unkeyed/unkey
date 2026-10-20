package httpclient

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type config struct {
	timeout   time.Duration
	attempts  int
	backoff   func(int) time.Duration
	transport http.RoundTripper
	breaker   circuitbreaker.CircuitBreaker[*http.Response]
	tracing   []otelhttp.Option
	headers   http.Header
}

// Option configures a client before it is used concurrently.
type Option func(*config)

// WithHeaders copies headers for each attempt without modifying the caller's request.
// Values are copied when this option is created. Multiple options merge headers,
// with later options replacing all values for the same canonical header name.
// Configured headers are removed on redirects with a different scheme or host
// (including port) from the initial request, to avoid forwarding credentials.
func WithHeaders(headers http.Header) Option {
	headers = headers.Clone()
	return func(c *config) {
		for name, values := range headers {
			c.headers[http.CanonicalHeaderKey(name)] = values
		}
	}
}

// WithTimeout bounds the entire call, including retries, redirects and body reads.
// Zero disables the timeout, as with [http.Client.Timeout].
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) { c.timeout = timeout }
}

// WithAttempts sets the total attempt limit, including the initial request.
// Values below one are treated as one, disabling retries.
func WithAttempts(attempts int) Option {
	return func(c *config) { c.attempts = max(1, attempts) }
}

// WithBackoff replaces the jittered exponential retry delay. The argument starts
// at one. The function must be safe for concurrent use and must not be nil.
func WithBackoff(backoff func(int) time.Duration) Option {
	return func(c *config) { c.backoff = backoff }
}

// WithTransport sets the underlying transport, for example an SSRF-safe dialer.
// It must be non-nil and safe for concurrent use.
func WithTransport(transport http.RoundTripper) Option {
	return func(c *config) { c.transport = transport }
}

// WithCircuitBreaker replaces the client-scoped breaker. It must not be nil.
func WithCircuitBreaker(breaker circuitbreaker.CircuitBreaker[*http.Response]) Option {
	return func(c *config) { c.breaker = breaker }
}

// WithTracing customizes OpenTelemetry instrumentation. By default it uses the
// global tracer provider and W3C trace-context propagation.
func WithTracing(options ...otelhttp.Option) Option {
	return func(c *config) { c.tracing = append(c.tracing, options...) }
}
