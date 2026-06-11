package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
)

// forwardConfig bundles the per-attempt inputs to forward. targetURL is the
// upstream endpoint, transport selects the RoundTripper (HTTP/1.1 vs h2c vs
// peer-region), and directorFunc applies header / hop adjustments specific
// to the destination kind (instance vs region peer). destination is a stable
// label used in metrics — for instance hops it is the deployment's host,
// for region hops it is "region.platform".
type forwardConfig struct {
	targetURL    *url.URL
	startTime    time.Time
	directorFunc func(*http.Request)
	destination  string
	transport    http.RoundTripper
}

// frontlineScopeAttrs is shared across requests: timing entries only read
// the map, so a single instance avoids a per-entry allocation on the hot
// path.
var frontlineScopeAttrs = map[string]string{"scope": "frontline"}

// proxyState is the per-request state for a single forward attempt. It
// replaces the closures that httputil.ReverseProxy's Director /
// ModifyResponse / ErrorHandler callbacks used to capture: instead of
// rebuilding the ReverseProxy (and three closures) on every request, one
// shared ReverseProxy reads this struct from the request context. That turns
// ~9 per-request heap allocations (the proxy struct, three closures, and the
// locals they forced to escape) into a single proxyState allocation.
//
// The captured response bytes (buf) are handed to the session / tracking
// record by reference after the request completes, so proxyState must not be
// pooled — reuse would corrupt an in-flight ClickHouse log.
type proxyState struct {
	svc         *service
	sess        *zen.Session
	cfg         forwardConfig
	tracking    *RequestTracking
	hasTracking bool

	// proxyStartTime/backendStart are written by the Director callback when
	// the proxy dispatches the upstream call.
	proxyStartTime time.Time
	backendStart   time.Time

	// buf accumulates the response body for ClickHouse logging via TeeReader
	// (only when hasTracking).
	buf bytes.Buffer
}

// publishUpstream records the upstream call duration once per request.
// Idempotent: subsequent calls (e.g. ErrorHandler after ModifyResponse) are
// no-ops because backendStart resets to zero.
func (ps *proxyState) publishUpstream(end time.Time) {
	if ps.backendStart.IsZero() {
		return
	}
	upstreamSeconds.WithLabelValues(ps.cfg.destination).Observe(end.Sub(ps.backendStart).Seconds())
	ps.backendStart = time.Time{}
}

// proxyStateCtxKey carries the per-request [proxyState] through the request
// context so the shared ReverseProxy's callbacks (and the transport shim)
// can reach it without per-request closures.
//
// It is a zero-size type rather than a [zen.ContextKey]: that helper holds a
// name string, so passing it by value to context.WithValue / ctx.Value boxes
// a >word struct into an interface and allocates on every call. An empty
// struct boxes to the shared runtime.zerobase pointer, so the only allocation
// left on this hot path is the unavoidable context node itself.
type proxyStateCtxKey struct{}

func withProxyState(ctx context.Context, ps *proxyState) context.Context {
	return context.WithValue(ctx, proxyStateCtxKey{}, ps)
}

func proxyStateFromContext(ctx context.Context) (*proxyState, bool) {
	ps, ok := ctx.Value(proxyStateCtxKey{}).(*proxyState)
	return ps, ok
}

// contextRoundTripper lets the single shared ReverseProxy dispatch to the
// correct upstream transport (http1 / h2c / peer-region) per request. The
// transport is chosen by the caller and stored on the request's proxyState;
// the shim itself is zero-size and shared.
type contextRoundTripper struct{}

func (contextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ps, ok := proxyStateFromContext(req.Context())
	if !ok || ps.cfg.transport == nil {
		return nil, fault.New("proxy state missing from request context")
	}
	return ps.cfg.transport.RoundTrip(req)
}

// newReverseProxy builds the one ReverseProxy shared by every request. Its
// callbacks are method values bound to the service (allocated once here, not
// per request) and recover per-request state from the request context.
func (s *service) newReverseProxy() *httputil.ReverseProxy {
	//nolint:exhaustruct
	return &httputil.ReverseProxy{
		Transport:  contextRoundTripper{},
		BufferPool: &s.copyBufs,
		// 0, not -1: ReverseProxy.flushInterval(res) overrides the value to
		// "flush immediately" whenever the response is streaming
		// (ContentLength == -1 or Content-Type: text/event-stream), so SSE
		// and chunked responses still flush per chunk. For known-length
		// responses, -1 forced a maxLatencyWriter + time.AfterFunc timer
		// per request and a flush syscall per body write, defeating
		// net/http's write coalescing for no benefit. 0 skips all of that
		// and lets the server flush once on handler return.
		FlushInterval:  0,
		Director:       s.proxyDirector,
		ModifyResponse: s.proxyModifyResponse,
		ErrorHandler:   s.proxyErrorHandler,
	}
}

func (s *service) proxyDirector(req *http.Request) {
	ps, ok := proxyStateFromContext(req.Context())
	if !ok {
		return
	}

	now := s.clock.Now()
	ps.proxyStartTime = now
	ps.backendStart = now
	if ps.hasTracking {
		ps.tracking.InstanceStart = now
	}

	req.URL.Scheme = ps.cfg.targetURL.Scheme
	req.URL.Host = ps.cfg.targetURL.Host

	ps.cfg.directorFunc(req)
}

func (s *service) proxyModifyResponse(resp *http.Response) error {
	ps, ok := proxyStateFromContext(resp.Request.Context())
	if !ok {
		return nil
	}

	// Success path: write timing headers here, before the proxy flushes
	// response headers to the client. The error path writes them after
	// ServeHTTP instead — never both.
	writeTimingHeaders(ps.sess.ResponseWriter(), s.clock.Now(), ps.cfg.startTime, ps.proxyStartTime)

	// Capture response body for logging via TeeReader.
	// Streaming: bytes flow to the client while accumulating in ps.buf.
	// Non-streaming: the proxy buffers internally, same result.
	//
	// Only when a tracking record exists: without one (ClickHouse logging
	// disabled or cross-region hop) nothing consumes the captured bytes, so
	// the tee would be pure overhead.
	//
	// Skip the wrap on protocol upgrades (101 Switching Protocols).
	// httputil.ReverseProxy.handleUpgradeResponse needs resp.Body to satisfy
	// io.ReadWriteCloser to hijack the upstream conn for bidirectional copy;
	// io.NopCloser would erase the Write half and break WebSockets. Logging a
	// "response body" is meaningless after an upgrade — the bytes that follow
	// are WS frames.
	if ps.hasTracking && resp.Body != nil && resp.StatusCode != http.StatusSwitchingProtocols {
		ps.buf.Reset()
		// Size the capture buffer from Content-Length so a known-length
		// response allocates once instead of growing by repeated doubling.
		ps.buf.Grow(zen.CaptureBufferHint(resp.ContentLength))
		resp.Body = io.NopCloser(io.TeeReader(resp.Body, &zen.LimitedWriter{W: &ps.buf, N: zen.MaxBodyCapture}))
	}

	return nil
}

func (s *service) proxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	ps, ok := proxyStateFromContext(r.Context())
	if !ok {
		return
	}

	ps.publishUpstream(s.clock.Now())

	// Capture the error for middleware to handle.
	if ecw, ok := w.(*zen.ErrorCapturingWriter); ok {
		ecw.SetError(err)

		logger.Warn(fmt.Sprintf("proxy error forwarding to %s", ps.cfg.destination),
			"error", err.Error(),
			"target", ps.cfg.targetURL.String(),
			"hostname", r.Host,
		)
	}
}

// writeTimingHeaders emits the frontline/total timing entries. It must run
// before response headers are flushed to count for the wire — i.e. inside
// ModifyResponse on the success path, or before the error middleware writes
// the response on the failure path.
func writeTimingHeaders(w http.ResponseWriter, now, startTime, proxyStartTime time.Time) {
	if !proxyStartTime.IsZero() {
		timing.Write(w, timing.Entry{
			Name:       "frontline",
			Duration:   proxyStartTime.Sub(startTime),
			Attributes: frontlineScopeAttrs,
		})
	}
	timing.Write(w, timing.Entry{
		Name:       "total",
		Duration:   now.Sub(startTime),
		Attributes: frontlineScopeAttrs,
	})
}

func (s *service) forward(ctx context.Context, sess *zen.Session, cfg forwardConfig) error {
	sess.ResponseWriter().Header().Set(HeaderFrontlineID, s.instanceID)
	sess.ResponseWriter().Header().Set(HeaderRegion, s.regionHeader)
	sess.ResponseWriter().Header().Set(HeaderRequestID, sess.RequestID())

	tracking, hasTracking := RequestTrackingFromContext(ctx)

	// One allocation per request carries everything the shared ReverseProxy's
	// callbacks need; see [proxyState]. proxyStartTime/backendStart/buf are
	// intentionally zero — the Director/ModifyResponse callbacks fill them in.
	//nolint:exhaustruct
	ps := &proxyState{
		svc:         s,
		sess:        sess,
		cfg:         cfg,
		tracking:    tracking,
		hasTracking: hasTracking,
	}

	wrapper := zen.NewErrorCapturingWriter(sess.ResponseWriter())

	// Proxy the request with the middleware context (carries timeout deadline)
	// plus the per-request proxy state the callbacks read back.
	reqCtx := withProxyState(ctx, ps)
	serveWithAbortRecovery(s.reverseProxy, wrapper, sess.Request().WithContext(reqCtx))

	// Record upstream duration after the full response (including streaming
	// body / WebSocket tunnel) has finished. Idempotent — if ErrorHandler
	// already published from the pre-response failure path, this is a no-op.
	ps.publishUpstream(s.clock.Now())

	// Mark the true end of the upstream interaction (full stream completed).
	if hasTracking {
		tracking.InstanceEnd = s.clock.Now()
		if ps.buf.Len() > 0 {
			tracking.ResponseBody = ps.buf.Bytes()
		}
	}

	// Feed captured response body back into the session for zen middleware logging.
	if ps.buf.Len() > 0 {
		sess.SetResponseBody(ps.buf.Bytes())
	}

	// If error was captured, return it to middleware for consistent error handling
	if err := wrapper.Error(); err != nil {
		// Error path: ModifyResponse never ran, so the timing headers have
		// not been written yet. The error middleware writes the response
		// after we return, so headers added here still reach the wire.
		writeTimingHeaders(sess.ResponseWriter(), s.clock.Now(), cfg.startTime, ps.proxyStartTime)
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
