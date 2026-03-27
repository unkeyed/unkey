package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
)

type forwardConfig struct {
	targetURL    *url.URL
	startTime    time.Time
	directorFunc func(*http.Request)
	destination  string
	transport    http.RoundTripper
}

func (s *service) forward(ctx context.Context, sess *zen.Session, cfg forwardConfig) error {
	sess.ResponseWriter().Header().Set(HeaderFrontlineID, s.instanceID)
	sess.ResponseWriter().Header().Set(HeaderRegion, fmt.Sprintf("%s::%s", s.platform, s.region))
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	var proxyStartTime time.Time

	defer func() {
		totalTime := s.clock.Now().Sub(cfg.startTime)
		if !proxyStartTime.IsZero() {
			timing.Write(sess.ResponseWriter(), timing.Entry{
				Name:     "frontline",
				Duration: proxyStartTime.Sub(cfg.startTime),
				Attributes: map[string]string{
					"scope": "frontline",
				},
			})
		}
		timing.Write(sess.ResponseWriter(), timing.Entry{
			Name:     "total",
			Duration: totalTime,
			Attributes: map[string]string{
				"scope": "frontline",
			},
		})
	}()

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	var backendStart time.Time
	observeBackendDuration := func() {
		if !backendStart.IsZero() {
			proxyBackendDuration.WithLabelValues(cfg.destination).Observe(s.clock.Now().Sub(backendStart).Seconds())
		}
	}

	// nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		Transport:     cfg.transport,
		FlushInterval: -1, // flush immediately for streaming
		Director: func(req *http.Request) {
			proxyStartTime = s.clock.Now()
			backendStart = proxyStartTime

			req.URL.Scheme = cfg.targetURL.Scheme
			req.URL.Host = cfg.targetURL.Host

			cfg.directorFunc(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			observeBackendDuration()

			source := "upstream"
			if resp.Header.Get("X-Unkey-Error-Source") == "sentinel" {
				source = "sentinel"
			}
			proxyBackendResponseTotal.WithLabelValues(cfg.destination, source, statusClass(resp.StatusCode)).Inc()

			totalTime := s.clock.Now().Sub(cfg.startTime)
			if !proxyStartTime.IsZero() {
				timing.Write(sess.ResponseWriter(), timing.Entry{
					Name:     "frontline",
					Duration: proxyStartTime.Sub(cfg.startTime),
					Attributes: map[string]string{
						"scope": "frontline",
					},
				})
			}
			timing.Write(sess.ResponseWriter(), timing.Entry{
				Name:     "total",
				Duration: totalTime,
				Attributes: map[string]string{
					"scope": "frontline",
				},
			})

			if source != "sentinel" {
				return nil
			}

			if sentinelTime := resp.Header.Get(timing.HeaderName); sentinelTime != "" {
				sess.ResponseWriter().Header().Add(timing.HeaderName, sentinelTime)
			}

			// 5xx from sentinel → fault error → frontline observability handles content negotiation
			if resp.StatusCode >= 500 {
				proxyForwardTotal.WithLabelValues(cfg.destination, "backend_5xx").Inc()

				// Try to extract the original error code from sentinel's JSON response
				// so we preserve the specific error (e.g. InvalidConfiguration → 500)
				// instead of blindly mapping everything to 502.
				urn, publicMessage := extractSentinelError(resp, resp.StatusCode)

				return fault.New(
					fmt.Sprintf("sentinel returned %d", resp.StatusCode),
					fault.Code(urn),
					fault.Public(publicMessage),
				)
			}

			// 4xx from sentinel (auth errors, rate limits) → rewrite to HTML if client prefers it,
			// otherwise pass the JSON through untouched.
			if resp.StatusCode >= 400 && wantsHTML(sess.Request()) {
				return rewriteSentinelErrorAsHTML(resp, sess.RequestID(), s.errorPageRenderer)
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			observeBackendDuration()

			// Capture the error for middleware to handle
			if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
				ecw.SetError(err)

				logger.Warn(fmt.Sprintf("proxy error forwarding to %s", cfg.destination),
					"error", err.Error(),
					"target", cfg.targetURL.String(),
					"hostname", r.Host,
				)
			}
		},
	}

	// Proxy the request with the middleware context (carries timeout deadline)
	proxy.ServeHTTP(wrapper, sess.Request().WithContext(ctx))

	// If error was captured, return it to middleware for consistent error handling
	if err := wrapper.Error(); err != nil {
		// If the error already has a fault code (e.g. from extractSentinelError
		// in ModifyResponse), preserve it instead of overwriting with a generic
		// proxy error.
		if _, hasCode := fault.GetCode(err); hasCode {
			return err
		}
		proxyForwardTotal.WithLabelValues(cfg.destination, categorizeProxyErrorType(err)).Inc()
		urn, message := categorizeProxyError(err, cfg.destination)
		return fault.Wrap(err,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to %s %s", cfg.destination, cfg.targetURL.String())),
			fault.Public(message),
		)
	}

	proxyForwardTotal.WithLabelValues(cfg.destination, "none").Inc()

	return nil
}

// categorizeProxyErrorType returns a short label for the type of proxy error,
// suitable for use as a prometheus label value.
func categorizeProxyErrorType(err error) string {
	if errors.Is(err, context.Canceled) {
		return "client_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return "timeout"
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return "conn_refused"
		}
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return "conn_reset"
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_failure"
	}

	return "other"
}

// wantsHTML returns true if the client prefers HTML over JSON based on the Accept header.
func wantsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch mediaType {
		case "text/html":
			return true
		case "application/json", "application/*", "*/*":
			return false
		}
	}

	return false
}

// sentinelError matches the JSON error structure returned by sentinel.
type sentinelError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// rewriteSentinelErrorAsHTML reads the sentinel JSON error response and replaces
// the body with a styled HTML error page. The original status code is preserved.
func rewriteSentinelErrorAsHTML(resp *http.Response, requestID string, renderer errorpage.Renderer) error {
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil // can't read body, let it pass through
	}

	var parsed sentinelError
	if err := json.Unmarshal(body, &parsed); err != nil {
		// Not valid JSON, put the body back unchanged
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	message := parsed.Error.Message
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	title := http.StatusText(resp.StatusCode)
	if title == "" {
		title = "Error"
	}

	var docsURL string
	if parsed.Error.Code != "" {
		if code, parseErr := codes.ParseCode(parsed.Error.Code); parseErr == nil {
			docsURL = code.DocsURL()
		}
	}

	htmlBody, renderErr := renderer.Render(errorpage.Data{
		StatusCode: resp.StatusCode,
		Title:      title,
		Message:    message,
		ErrorCode:  parsed.Error.Code,
		DocsURL:    docsURL,
		RequestID:  requestID,
	})
	if renderErr != nil {
		// Template render failed, put original body back
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(htmlBody))
	resp.ContentLength = int64(len(htmlBody))
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(htmlBody)))

	return nil
}

// extractSentinelError reads sentinel's JSON error response to extract the original
// error code and message. This preserves sentinel's specific error codes (e.g.
// InvalidConfiguration → 500) instead of replacing them all with BadGateway (502).
// Falls back to generic frontline codes based on HTTP status when the body can't be parsed.
func extractSentinelError(resp *http.Response, statusCode int) (codes.URN, string) {
	fallbackURN := codes.Frontline.Proxy.BadGateway.URN()
	switch statusCode {
	case http.StatusServiceUnavailable:
		fallbackURN = codes.Frontline.Proxy.ServiceUnavailable.URN()
	case http.StatusGatewayTimeout:
		fallbackURN = codes.Frontline.Proxy.GatewayTimeout.URN()
	case http.StatusBadGateway:
		fallbackURN = codes.Frontline.Proxy.BadGateway.URN()
	}
	fallbackMessage := http.StatusText(statusCode)

	if resp.Body == nil {
		return fallbackURN, fallbackMessage
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil || len(body) == 0 {
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return fallbackURN, fallbackMessage
	}

	// Put body back so downstream can still read it
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var parsed sentinelError
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fallbackURN, fallbackMessage
	}

	if parsed.Error.Code != "" {
		urn := codes.URN(parsed.Error.Code)
		message := parsed.Error.Message
		if message == "" {
			message = fallbackMessage
		}
		return urn, message
	}

	return fallbackURN, fallbackMessage
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
