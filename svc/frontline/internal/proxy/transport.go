package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"golang.org/x/net/http2"
)

// TransportRegistry holds pre-initialized transports for each supported
// upstream protocol. The correct transport is selected per-request based on
// the deployment's configured protocol. Connections are pooled per registry,
// so callers must reuse a single registry across requests.
type TransportRegistry struct {
	transports map[db.DeploymentsUpstreamProtocol]http.RoundTripper
	fallback   http.RoundTripper
}

// countingDialContext wraps a dial function so every new TCP connection
// increments upstreamDialsTotal. Counting at the transport level costs
// nothing on pooled-connection requests, unlike the previous per-request
// httptrace.ClientTrace which allocated a trace struct + context on every
// request to observe an event that only happens on (rare) new dials.
func countingDialContext(dial func(context.Context, string, string) (net.Conn, error), destination string) func(context.Context, string, string) (net.Conn, error) {
	success := upstreamDialsTotal.WithLabelValues(destination, dialOutcomeSuccess)
	failure := upstreamDialsTotal.WithLabelValues(destination, dialOutcomeError)
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dial(ctx, network, addr)
		if err != nil {
			failure.Inc()
		} else {
			success.Inc()
		}
		return conn, err
	}
}

// NewTransportRegistry creates transports for http1 and h2c. Unknown or
// unimplemented protocols (h2, h3) fall back to http1.
func NewTransportRegistry() *TransportRegistry {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	} //nolint:exhaustruct

	//nolint:exhaustruct
	h1 := &http.Transport{
		DialContext: countingDialContext(dialer.DialContext, destinationInstance),
		// All requests for a deployment in this region hit a handful of
		// instance addresses, so per-host is effectively the pool size.
		// Sized for high-concurrency load: a small per-host cap forces
		// connection churn (dial + slow-start per request) once concurrent
		// requests exceed it.
		MaxIdleConns:        1024,
		MaxIdleConnsPerHost: 512,
		IdleConnTimeout:     90 * time.Second,
		// A proxy must not negotiate gzip on behalf of clients: with
		// DisableCompression unset, the transport silently adds
		// Accept-Encoding gzip when the client sent none and then
		// decompresses the response, burning CPU to undo work the
		// upstream just did. Pass encodings through untouched.
		DisableCompression: true,
		// Default read/write buffers are 4 KiB; typical API responses are
		// larger, and bigger buffers mean fewer syscalls per response.
		// These buffers live per connection (a 16 KiB reader + 16 KiB
		// writer = 32 KiB per conn), so the size trades resident memory
		// against syscall count — 64 KiB measured ~no extra throughput
		// over 16 KiB but 4x the idle-pool memory.
		ReadBufferSize:  16 << 10,
		WriteBufferSize: 16 << 10,
	}

	h2cDial := countingDialContext(dialer.DialContext, destinationInstance)
	//nolint:exhaustruct
	h2c := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return h2cDial(ctx, network, addr)
		},
		// Same reasoning as the h1 transport: pass encodings through.
		DisableCompression: true,
	}

	return &TransportRegistry{
		transports: map[db.DeploymentsUpstreamProtocol]http.RoundTripper{
			db.DeploymentsUpstreamProtocolHttp1: h1,
			db.DeploymentsUpstreamProtocolH2c:   h2c,
		},
		fallback: h1,
	}
}

// Get returns the transport for the given protocol. Falls back to http1 for
// unknown or unimplemented protocols.
func (r *TransportRegistry) Get(proto db.DeploymentsUpstreamProtocol) http.RoundTripper {
	if t, ok := r.transports[proto]; ok {
		return t
	}
	logger.Warn("unsupported upstream protocol, falling back to http1", "protocol", string(proto))
	return r.fallback
}
