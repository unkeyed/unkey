package providers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/ssrf"
)

const (
	requestTimeout    = 10 * time.Second
	responseMax       = 1 << 20
	retryAfterMax     = time.Hour
	retryAfterDefault = time.Second
)

func newHTTPClient() *http.Client {
	return ssrf.New(ssrf.WithTimeout(requestTimeout))
}

func sendError(kind notification.FailureKind, code string) *notification.SendError {
	return &notification.SendError{Kind: kind, Code: code, RetryAfter: 0}
}

func validateURL(raw string) (*url.URL, error) {
	if err := ssrf.ValidateEndpoint(raw); err != nil {
		return nil, errors.New("invalid webhook URL")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" {
		return nil, errors.New("invalid webhook URL")
	}
	return u, nil
}

func send(client *http.Client, req *http.Request) ([]byte, int, http.Header, error) {
	var connecting, connected atomic.Bool
	var trace httptrace.ClientTrace
	trace.GetConn = func(string) { connecting.Store(true) }
	trace.GotConn = func(httptrace.GotConnInfo) { connected.Store(true) }
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &trace))
	res, err := client.Do(req)
	if err != nil {
		if !connected.Load() {
			if errors.Is(err, ssrf.ErrForbiddenIP) {
				return nil, 0, nil, sendError(notification.Rejected, "forbidden_destination")
			}
			var dnsError *net.DNSError
			var networkError *net.OpError
			if errors.As(err, &dnsError) || (errors.As(err, &networkError) && networkError.Op == "dial") {
				return nil, 0, nil, sendError(notification.Retryable, "connection_failed")
			}
			var timeout net.Error
			if connecting.Load() && errors.As(err, &timeout) && timeout.Timeout() {
				return nil, 0, nil, sendError(notification.Retryable, "connection_timeout")
			}
		}
		return nil, 0, nil, sendError(notification.Ambiguous, "request_failed")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, responseMax+1))
	if err != nil || len(body) > responseMax {
		return nil, res.StatusCode, res.Header, sendError(notification.Ambiguous, "invalid_response")
	}
	return body, res.StatusCode, res.Header, nil
}

func statusError(status int, header http.Header) error {
	if status == http.StatusTooManyRequests {
		delay, ok := parseRetryAfter(header.Get("Retry-After"), time.Now())
		if !ok {
			delay = retryAfterDefault
		}
		return &notification.SendError{Kind: notification.Retryable, Code: "rate_limited", RetryAfter: delay}
	}
	if status == http.StatusRequestTimeout {
		return sendError(notification.Ambiguous, "request_timeout")
	}
	if status >= 400 && status < 500 {
		return sendError(notification.Rejected, "request_rejected")
	}
	return sendError(notification.Ambiguous, "provider_unavailable")
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	var delay time.Duration
	if err == nil && seconds >= 0 {
		if seconds > int64(retryAfterMax/time.Second) {
			return retryAfterMax, true
		}
		delay = time.Duration(seconds) * time.Second
	} else if at, dateErr := http.ParseTime(value); dateErr == nil {
		delay = at.Sub(now)
		if delay < 0 {
			delay = 0
		}
	} else {
		return 0, false
	}
	if delay > retryAfterMax {
		delay = retryAfterMax
	}
	return delay, true
}

func newJSONRequest(ctx context.Context, endpoint string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("invalid request")
	}
	req.GetBody = nil
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
