package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
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

	// publishUpstream records the upstream call duration once per request.
	// Idempotent: subsequent calls (e.g. ErrorHandler after ModifyResponse)
	// are no-ops because backendStart resets to zero.
	publishUpstream := func(end time.Time) {
		if backendStart.IsZero() {
			return
		}
		upstreamSeconds.WithLabelValues(cfg.destination).Observe(end.Sub(backendStart).Seconds())
		backendStart = time.Time{}
	}

	// nolint:exhaustruct
	clientTrace := &httptrace.ClientTrace{
		ConnectDone: func(network, addr string, err error) {
			outcome := dialOutcomeSuccess
			if err != nil {
				outcome = dialOutcomeError
			}
			upstreamDialsTotal.WithLabelValues(cfg.destination, outcome).Inc()
		},
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

			*req = *req.WithContext(httptrace.WithClientTrace(req.Context(), clientTrace))
		},
		ModifyResponse: func(resp *http.Response) error {
			publishUpstream(s.clock.Now())

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
			//
			// Skip the wrap on protocol upgrades (101 Switching Protocols).
			// httputil.ReverseProxy.handleUpgradeResponse needs resp.Body to
			// satisfy io.ReadWriteCloser to hijack the upstream conn for
			// bidirectional copy; io.NopCloser would erase the Write half
			// and break WebSockets. Logging a "response body" is meaningless
			// after an upgrade — the bytes that follow are WS frames.
			if resp.Body != nil && resp.StatusCode != http.StatusSwitchingProtocols {
				responseBuf.Reset()
				resp.Body = io.NopCloser(io.TeeReader(resp.Body, &zen.LimitedWriter{W: &responseBuf, N: zen.MaxBodyCapture}))
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			publishUpstream(s.clock.Now())

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
	serveWithAbortRecovery(proxy, wrapper, sess.Request().WithContext(ctx))

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
		urn, message := categorizeProxyError(err, cfg.destination)
		return fault.Wrap(err,
			fault.Code(urn),
			fault.Internal(fmt.Sprintf("proxy error forwarding to %s %s", cfg.destination, cfg.targetURL.String())),
			fault.Public(message),
		)
	}

	return nil
}

// serveWithAbortRecovery calls h.ServeHTTP and swallows http.ErrAbortHandler panics.
// httputil.ReverseProxy panics with that sentinel value when the response body copy fails
// after headers have been flushed (typically the client disconnected mid-stream).
// There is no recovery path at that point — the panic is the library's signal that
// the handler is aborting. Swallowing it keeps the global panic middleware strict
// for real bugs; other panic values are re-raised unchanged.
//
// Aborts after headers were already flushed render as ordinary 2xx successes
// in requests_total — the response we committed to was just truncated.
// Aborts before headers go through the proxy's ErrorHandler with a
// User.BadRequest.ClientClosedRequest URN.
func serveWithAbortRecovery(h http.Handler, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			if rec != http.ErrAbortHandler {
				panic(rec)
			}
		}
	}()
	h.ServeHTTP(w, r)
}
