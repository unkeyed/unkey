package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Doer is the single method httpx needs from *http.Client. Tests can
// substitute a recording stub without dragging in net/http/httptest.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Option mutates the request before it is sent. Use [Header],
// [Bearer], or compose your own.
type Option func(*http.Request)

// Header sets a request header, replacing any existing value.
func Header(key, value string) Option {
	return func(r *http.Request) { r.Header.Set(key, value) }
}

// Bearer is the common case of Authorization: Bearer <token>.
func Bearer(token string) Option {
	return Header("Authorization", "Bearer "+token)
}

// WithDoer overrides the default *http.Client for one call. Mostly
// useful in tests; production code should prefer building a single
// Doer at startup and reusing it for connection pooling.
type doerKey struct{}

func WithDoer(d Doer) Option {
	return func(r *http.Request) { *r = *r.WithContext(context.WithValue(r.Context(), doerKey{}, d)) }
}

// Get sends GET url and decodes the JSON response into Out.
func Get[Out any](ctx context.Context, url string, opts ...Option) (Out, error) {
	return do[Out](ctx, http.MethodGet, url, nil, opts)
}

// Post sends POST url with in encoded as JSON and decodes the
// response into Out. Use [Empty] for In or Out when one side has no
// JSON body.
func Post[In, Out any](ctx context.Context, url string, in In, opts ...Option) (Out, error) {
	return do[Out](ctx, http.MethodPost, url, &in, opts)
}

// Put sends PUT url with in encoded as JSON and decodes the response
// into Out.
func Put[In, Out any](ctx context.Context, url string, in In, opts ...Option) (Out, error) {
	return do[Out](ctx, http.MethodPut, url, &in, opts)
}

// Delete sends DELETE url and decodes the JSON response into Out.
func Delete[Out any](ctx context.Context, url string, opts ...Option) (Out, error) {
	return do[Out](ctx, http.MethodDelete, url, nil, opts)
}

// Empty is the type to use when a Get/Post/Put/Delete has no
// meaningful response body. The decoded value is the zero struct;
// any response body is discarded.
type Empty struct{}

// StatusError is returned for any response with status outside 2xx.
// Body is the raw response body (capped at 8 KiB) so the caller can
// log details without needing to re-read.
type StatusError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *StatusError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("httpx: %s", e.Status)
	}
	return fmt.Sprintf("httpx: %s: %s", e.Status, bytes.TrimSpace(e.Body))
}

// IsStatus reports whether err is a *StatusError matching code.
func IsStatus(err error, code int) bool {
	var se *StatusError
	if !errors.As(err, &se) {
		return false
	}
	return se.StatusCode == code
}

// maxErrorBody caps the bytes we capture in StatusError.Body so a
// 200 MiB error response cannot pin a probe runner's heap.
const maxErrorBody = 8 << 10

// do is the shared implementation for Get/Post/Put/Delete. body is a
// pointer-to-In so the JSON encoder sees a typed value (and a nil
// pointer means "no body"); In is generic, so we cannot accept
// `body In` and check for nil.
func do[Out any](ctx context.Context, method, url string, body any, opts []Option) (Out, error) {
	var zero Out

	var reqBody io.Reader
	hasBody := false
	if body != nil {
		// body is *In where In is a generic type; marshal the
		// dereferenced value.
		raw, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("httpx: marshal request: %w", err)
		}
		// Convention: for In = struct{} (i.e. Empty), skip the body
		// so we send a real bodyless request. JSON-encoded `{}` is
		// not the same thing semantically.
		if string(raw) != "{}" {
			reqBody = bytes.NewReader(raw)
			hasBody = true
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return zero, fmt.Errorf("httpx: build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, opt := range opts {
		opt(req)
	}

	doer, _ := req.Context().Value(doerKey{}).(Doer)
	if doer == nil {
		doer = http.DefaultClient
	}

	resp, err := doer.Do(req)
	if err != nil {
		return zero, fmt.Errorf("httpx: %s %s: %w", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode/100 != 2 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return zero, &StatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       bodyBytes,
		}
	}

	// Empty out type: caller doesn't want the body. Drain to keep
	// the connection reusable, then return zero value.
	if _, ok := any(zero).(Empty); ok {
		_, _ = io.Copy(io.Discard, resp.Body)
		return zero, nil
	}

	var out Out
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return zero, fmt.Errorf("httpx: decode %s response: %w", url, err)
	}
	return out, nil
}
