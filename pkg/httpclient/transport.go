package httpclient

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/retry"
)

type transport struct {
	config config
	next   http.RoundTripper
}

var errRetryStatus = errors.New("retryable HTTP status")

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	started := false
	res, err := t.config.breaker.Do(req.Context(), func(ctx context.Context) (*http.Response, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		started = true
		res, err := t.sendWithRetries(req)
		if err == nil && res.StatusCode >= 500 {
			return res, errRetryStatus
		}
		return res, err
	})
	if !started && req.Body != nil {
		err = errors.Join(err, req.Body.Close())
	}
	if errors.Is(err, errRetryStatus) && res != nil {
		return res, nil
	}
	return res, err
}

func (t *transport) sendWithRetries(req *http.Request) (*http.Response, error) {
	attempts := t.config.attempts
	if !replayable(req) {
		attempts = 1
	}
	attempt := 0
	var retryAfter time.Duration
	retries := retry.New(
		retry.Attempts(attempts),
		retry.ShouldRetry(transient),
		retry.Backoff(func(n int) time.Duration {
			return max(t.config.backoff(n), retryAfter)
		}),
	)
	res, err := retries.DoWithResultContext(req.Context(), func() (*http.Response, error) {
		attempt++
		retryAfter = 0
		res, err := t.sendAttempt(req, attempt)
		if err != nil {
			return nil, err
		}
		if attempt == attempts || !retryStatus(res.StatusCode) {
			return res, nil
		}
		retryAfter = retryDelay(res.Header.Get("Retry-After"))
		if err := res.Body.Close(); err != nil {
			return nil, err
		}
		return nil, errRetryStatus
	})
	if attempt == 0 && req.Body != nil {
		err = errors.Join(err, req.Body.Close())
	}
	return res, err
}

func (t *transport) sendAttempt(req *http.Request, attempt int) (*http.Response, error) {
	current := req.Clone(req.Context())
	original := req
	for original.Response != nil {
		original = original.Response.Request
	}
	for name, values := range t.config.headers {
		if current.URL.Scheme == original.URL.Scheme && current.URL.Host == original.URL.Host {
			current.Header[name] = slices.Clone(values)
		} else {
			current.Header.Del(name)
		}
	}
	if attempt > 1 && req.Body != nil && req.Body != http.NoBody {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		current.Body = body
	}
	return t.next.RoundTrip(current)
}

func (t *transport) CloseIdleConnections() {
	if closer, ok := t.config.transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func replayable(req *http.Request) bool {
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return false
	}
	switch req.Method {
	case "", http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func retryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func transient(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var networkError net.Error
	return errors.Is(err, errRetryStatus) || errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) || errors.As(err, &networkError)
}

func retryDelay(value string) time.Duration {
	if seconds, err := strconv.ParseUint(value, 10, 64); err == nil {
		return time.Duration(min(seconds, uint64((1<<63-1)/time.Second))) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(0, time.Until(date))
	}
	return 0
}
