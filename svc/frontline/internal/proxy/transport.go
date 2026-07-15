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

// NewTransportRegistry creates transports for http1 and h2c. Unknown or
// unimplemented protocols (h2, h3) fall back to http1.
func NewTransportRegistry() *TransportRegistry {
	//nolint:exhaustruct
	h1 := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	}

	//nolint:exhaustruct
	h2c := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			d := net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return d.DialContext(ctx, network, addr)
		},
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
