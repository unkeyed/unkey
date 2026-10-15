package peer

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
)

// Relay serves backhaul streams in the instance's region. It does no route
// lookup and no policy evaluation: the entry frontline already did both and
// signed the target.
type Relay struct {
	Metadata *meta.Codec
	Targets  TargetValidator
	// Transport returns the upstream transport for a protocol ("http1",
	// "h2c"). Upgrades always use "http1".
	Transport func(protocol string) http.RoundTripper
	Clock     clock.Clock
}

// Server returns an h2c-only server for r with limits sized for long-lived
// streams over a high-latency link.
func (r *Relay) Server() *http.Server {
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	//nolint:exhaustruct
	return &http.Server{
		Handler:           r,
		Protocols:         &protocols,
		ReadHeaderTimeout: 10 * time.Second,
		//nolint:exhaustruct
		HTTP2: &http.HTTP2Config{
			MaxConcurrentStreams:          4096,
			MaxReceiveBufferPerConnection: 64 << 20,
			MaxReceiveBufferPerStream:     4 << 20,
			MaxReadFrameSize:              1 << 20,
			SendPingTimeout:               15 * time.Second,
			PingTimeout:                   5 * time.Second,
		},
	}
}

func (r *Relay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	target, reason := r.verify(req)
	if reason != "" {
		reject(w, reason)
		return
	}
	if err := r.Targets.Validate(target.DeploymentID, target.Address); err != nil {
		reject(w, rejectInstanceUnknown)
		return
	}
	req.Header.Del(HeaderFrontlineMeta)

	if req.Method == http.MethodConnect {
		r.upgrade(w, req, target)
		return
	}
	r.proxy(w, req, target)
}

func (r *Relay) verify(req *http.Request) (*meta.Target, string) {
	values := req.Header.Values(HeaderFrontlineMeta)
	if len(values) != 1 {
		return nil, rejectUnauthorized
	}
	md, err := r.Metadata.Unmarshal(values[0])
	if err != nil || md.ExpiresAt.IsZero() || !r.Clock.Now().Before(md.ExpiresAt) {
		return nil, rejectUnauthorized
	}
	if len(md.Hops) != 1 {
		return nil, rejectHops
	}
	if md.Target == nil || md.Target.Address == "" || md.Target.DeploymentID == "" {
		return nil, rejectBadRequest
	}
	return md.Target, ""
}

func reject(w http.ResponseWriter, reason string) {
	relayRejectsTotal.WithLabelValues(reason).Inc()
	w.Header().Set(headerReject, reason)
	w.WriteHeader(http.StatusServiceUnavailable)
}

// ackOnConn sends the dial acknowledgement once the upstream transport has a
// connection, before the request is written to the pod.
func ackOnConn(w http.ResponseWriter, acked *bool) *httptrace.ClientTrace {
	//nolint:exhaustruct
	return &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) {
			if !*acked {
				*acked = true
				w.WriteHeader(statusDialed)
			}
		},
	}
}

func (r *Relay) proxy(w http.ResponseWriter, req *http.Request, target *meta.Target) {
	removePeerHeaders(req.Header)
	acked := false
	//nolint:exhaustruct
	rp := &httputil.ReverseProxy{
		Transport:     r.Transport(target.Protocol),
		FlushInterval: -1,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = target.Address
			pr.Out.Host = pr.In.Host
			// Rewrite drops forwarding headers; the entry set them from the
			// client connection, so they are authoritative here.
			for _, name := range forwardingHeaders {
				if values := pr.In.Header.Values(name); len(values) > 0 {
					pr.Out.Header[name] = values
				}
			}
			*pr.Out = *pr.Out.WithContext(httptrace.WithClientTrace(pr.Out.Context(), ackOnConn(w, &acked)))
		},
		ModifyResponse: func(resp *http.Response) error {
			removePeerHeaders(resp.Header)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			if !acked {
				reject(w, rejectDialFailed)
				return
			}
			logger.Warn("backhaul relay upstream error", "target", target.Address, "error", err.Error())
			w.WriteHeader(http.StatusBadGateway)
		},
	}
	serve(rp, w, req)
}

func serve(h http.Handler, w http.ResponseWriter, req *http.Request) {
	defer func() {
		if rec := recover(); rec != nil && rec != http.ErrAbortHandler {
			panic(rec)
		}
	}()
	h.ServeHTTP(w, req)
}

func (r *Relay) upgrade(w http.ResponseWriter, req *http.Request, target *meta.Target) {
	token := req.Header.Get(headerUpgrade)
	requestTarget := req.Header.Get(headerRequestTarget)
	if token == "" || !strings.HasPrefix(requestTarget, "/") {
		reject(w, rejectBadRequest)
		return
	}
	upURL, err := url.ParseRequestURI(requestTarget)
	if err != nil {
		reject(w, rejectBadRequest)
		return
	}
	upURL.Scheme = "http"
	upURL.Host = target.Address

	header := req.Header.Clone()
	removePeerHeaders(header)
	removeHopHeaders(header)
	header.Set("Connection", "Upgrade")
	header.Set("Upgrade", token)

	// An upgrade has no body to replay, so it needs no dial ack; skipping it
	// saves a packet per socket on a connection that may be in slow start.
	dialed := false
	//nolint:exhaustruct
	ctx := httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { dialed = true },
	})
	//nolint:exhaustruct
	up := (&http.Request{
		Method:     http.MethodGet,
		URL:        upURL,
		Host:       req.Host,
		Header:     header,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}).WithContext(ctx)

	resp, err := r.Transport("http1").RoundTrip(up)
	if err != nil {
		if !dialed {
			reject(w, rejectDialFailed)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	upgradedTo := resp.Header.Get("Upgrade")
	removePeerHeaders(resp.Header)
	removeHopHeaders(resp.Header)
	for name, values := range resp.Header {
		w.Header()[name] = values
	}
	w.Header().Set(headerUpstreamStatus, strconv.Itoa(resp.StatusCode))
	w.Header().Set(headerUpgrade, upgradedTo)
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.Flush()

	out := &flushWriter{w: w, rc: rc}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_, _ = io.Copy(out, resp.Body)
		return
	}
	backend, ok := resp.Body.(io.ReadWriteCloser)
	if !ok {
		return
	}
	go func() {
		_, _ = io.Copy(backend, req.Body)
		_ = backend.Close()
	}()
	_, err = io.Copy(out, backend)
	if err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Debug("backhaul tunnel ended", "target", target.Address, "error", err.Error())
	}
}

var forwardingHeaders = []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"}

type flushWriter struct {
	w  io.Writer
	rc *http.ResponseController
}

func (f *flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if err == nil {
		err = f.rc.Flush()
	}
	return n, err
}
