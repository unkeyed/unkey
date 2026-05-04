package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
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

	var responseBuf bytes.Buffer

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	var backendStart time.Time
	observeBackendDuration := func() {
		if !backendStart.IsZero() {
			proxyBackendDuration.WithLabelValues(cfg.destination).Observe(s.clock.Now().Sub(backendStart).Seconds())
		}
	}

	tracking, hasTracking := RequestTrackingFromContext(ctx)

	// nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		Transport:     cfg.transport,
		FlushInterval: -1, // flush immediately for streaming
		Director: func(req *http.Request) {
			proxyStartTime = s.clock.Now()
			backendStart = proxyStartTime
			if hasTracking {
				tracking.InstanceStart = proxyStartTime
			}

			req.URL.Scheme = cfg.targetURL.Scheme
			req.URL.Host = cfg.targetURL.Host

			cfg.directorFunc(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			observeBackendDuration()

			// Both destinations stream upstream responses through unchanged.
			// "instance" hits a customer pod; "region" hits a peer frontline
			// that already wrote its own JSON/HTML error page via its
			// observability middleware. We just record + pass through.
			source := "upstream"
			proxyBackendResponseTotal.WithLabelValues(cfg.destination, source, statusClass(resp.StatusCode)).Inc()

			if hasTracking {
				tracking.ResponseStatus = int32(resp.StatusCode)
				tracking.ResponseHeaders = resp.Header
			}

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

			// Capture response body for logging via TeeReader.
			// Streaming: bytes flow to the client while accumulating in responseBuf.
			// Non-streaming: the proxy buffers internally, same result.
			if resp.Body != nil {
				responseBuf.Reset()
				resp.Body = io.NopCloser(io.TeeReader(resp.Body, &zen.LimitedWriter{W: &responseBuf, N: zen.MaxBodyCapture}))
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

	// Proxy the request with the middleware context (carries timeout deadline).
	serveWithAbortRecovery(proxy, wrapper, sess.Request().WithContext(ctx), cfg.destination)

	// Mark the true end of the upstream interaction (full stream completed).
	if hasTracking {
		tracking.InstanceEnd = s.clock.Now()
		if responseBuf.Len() > 0 {
			tracking.ResponseBody = responseBuf.Bytes()
		}
	}

	// Feed captured response body back into the session for zen middleware logging.
	if responseBuf.Len() > 0 {
		sess.SetResponseBody(responseBuf.Bytes())
	}

	// If error was captured, return it to middleware for consistent error handling
	if err := wrapper.Error(); err != nil {
		if _, hasCode := fault.GetCode(err); hasCode {
			return err
		}
		proxyForwardTotal.WithLabelValues(cfg.destination, categorizeProxyErrorType(err)).Inc()
		proxyForwardErrorsTotal.WithLabelValues(cfg.destination).Inc()
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

// serveWithAbortRecovery calls h.ServeHTTP and swallows http.ErrAbortHandler panics.
// httputil.ReverseProxy panics with that sentinel value when the response body copy fails
// after headers have been flushed (typically the client disconnected mid-stream).
// There is no recovery path at that point — the panic is the library's signal that
// the handler is aborting. Swallowing it keeps the global panic middleware strict
// for real bugs; other panic values are re-raised unchanged.
func serveWithAbortRecovery(h http.Handler, w http.ResponseWriter, r *http.Request, destination string) {
	defer func() {
		if rec := recover(); rec != nil {
			if rec != http.ErrAbortHandler {
				panic(rec)
			}
			proxyAbortedTotal.WithLabelValues(destination).Inc()
		}
	}()
	h.ServeHTTP(w, r)
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
