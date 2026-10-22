// Package httpclient creates standard HTTP clients with bounded calls,
// retries, circuit breaking, and OpenTelemetry instrumentation.
//
// Reuse one client per downstream service. Name it after the downstream hostname,
// such as api.resend.com, so circuit-breaker metrics use consistent labels across
// callers. Omit the scheme, port, path, query, and credentials. Avoid tenant-specific
// hostnames that would create high-cardinality labels. The name is only a label;
// clients with the same name still have independent circuit breakers:
//
//	client := httpclient.New("api.resend.com",
//		httpclient.WithTimeout(10*time.Second),
//		httpclient.WithAttempts(3),
//	)
//
// New returns an *http.Client, so SDKs can accept it without an adapter. Its
// transport implements http.RoundTripper. Keep the client when using its
// timeout: moving only its Transport to another client does not copy Timeout.
// Clients and transports are safe for concurrent use after configuration.
//
// The defaults are a 30-second overall timeout and at most three attempts per
// redirect hop. Retries use exponential full jitter starting at 100 milliseconds
// and capped at 5 seconds. Retry-After can extend the wait, but never the overall
// deadline. Retries apply to network errors, EOF, and HTTP 429, 502, 503, and 504.
// Only GET, HEAD, OPTIONS, TRACE, PUT, and DELETE are retried. A nonempty request
// body also needs GetBody; http.NewRequest supplies it for common buffer types.
// This retry layer does not retry POST or PATCH, even with an idempotency key.
// The underlying transport can have its own retries, as http.Transport does for
// some connection failures. WithAttempts bounds this layer, not those retries.
//
// The circuit breaker observes the final result of each retry sequence. Transport
// errors other than context cancellation, and all final 5xx responses, count as
// failures. Response-body read errors occur after this decision and do not count.
// The default breaker uses pkg/circuitbreaker's count and recovery policy.
// HTTP statuses remain responses, not Go errors. Always close the response body.
//
// Each attempt creates an OpenTelemetry client span under the request's parent
// context and propagates W3C trace context. Retry waits and circuit-breaker
// rejections do not create spans. Configure the application's global tracer
// provider to export spans, or supply one through WithTracing.
// Spans end on transport errors or when the body closes or reaches EOF.
//
// Redirects, proxies, and TLS use standard net/http behavior. This package does
// not protect against SSRF. Customer-supplied URLs need separate endpoint,
// dial, and redirect validation, such as the policies in pkg/ssrf.
package httpclient
