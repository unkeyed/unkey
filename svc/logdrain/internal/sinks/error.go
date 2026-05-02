package sinks

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// SendError is the typed error every Sink wraps its failures in. It lets
// the worker decide whether a Send is worth retrying without re-classifying
// HTTP status codes per provider, and surfaces the provider's
// Retry-After hint where one is present.
//
// Status == 0 means the request never reached the provider (DNS failure,
// connection reset, context deadline). Those are always retryable;
// idempotent POSTs at the same batch are safe to repeat.
type SendError struct {
	// Err is the underlying error, suitable for errors.Unwrap.
	Err error
	// Status is the HTTP status code from the provider, or 0 when the
	// request failed before getting a response.
	Status int
	// RetryAfter is the parsed Retry-After header value (or 0 if absent).
	// Honoured by the worker before falling back to the exponential
	// schedule.
	RetryAfter time.Duration
}

// Error implements error. The provider's verbatim message is preserved so
// the dashboard's failure timeline shows the actual reason.
func (e *SendError) Error() string {
	if e == nil || e.Err == nil {
		return "<nil send error>"
	}
	return e.Err.Error()
}

// Unwrap exposes the underlying error to errors.Is / errors.As.
func (e *SendError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Retryable reports whether the worker should attempt this Send again.
// Transport-level failures (Status == 0) are retryable. For HTTP errors:
// 5xx and the two retry-friendly 4xx codes (408 timeout, 429 rate limit)
// are retryable; everything else is fatal because retrying a 401/403/400
// just generates load with no chance of succeeding.
func (e *SendError) Retryable() bool {
	if e == nil {
		return false
	}

	if e.Status == 0 {
		return true
	}

	if e.Status >= 500 {
		return true
	}

	return e.Status == http.StatusRequestTimeout || e.Status == http.StatusTooManyRequests
}

// TransportError builds a SendError for failures that never reached the
// provider (DNS, connection refused, TLS, context cancellation).
func TransportError(provider string, err error) error {
	return &SendError{
		Err:        fmt.Errorf("%s: %w", provider, err),
		Status:     0,
		RetryAfter: 0,
	}
}

// HTTPError builds a SendError for non-2xx provider responses, parsing
// Retry-After as a side effect so the worker can honour it. The body
// excerpt is appended to the message for dashboard visibility.
func HTTPError(provider string, resp *http.Response, body []byte) error {
	return &SendError{
		Err:        fmt.Errorf("%s returned %d: %s", provider, resp.StatusCode, string(body)),
		Status:     resp.StatusCode,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

// parseRetryAfter handles both forms of Retry-After: an integer seconds
// count or an HTTP-date. Any unparseable value yields 0, which falls
// through to the worker's exponential backoff schedule.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}

	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}

	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// AsSendError extracts a *SendError from an error chain when present. For
// non-SendError values it returns a synthetic transport SendError so the
// worker can treat any unwrapped failure as "retryable, no Retry-After
// hint" without nil-checking on every call site.
func AsSendError(err error) *SendError {
	if err == nil {
		return nil
	}

	var se *SendError
	if errors.As(err, &se) {
		return se
	}

	// Unwrapped plain error from a sink that doesn't yet emit SendError.
	// Treat as transport failure so the worker still retries it; the
	// alternative (treating unknown errors as fatal) would silently drop
	// any sink we forgot to migrate.
	return &SendError{Err: err, Status: 0, RetryAfter: 0}
}
