package controlplane

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"golang.org/x/net/http2"
)

// ClientConfig holds the configuration for creating a control plane client.
//
// All fields are required for proper client operation. The client will automatically
// add authentication and routing metadata to all requests through an interceptor.
type ClientConfig struct {
	// URL is the base URL of the control plane service.
	URL string

	// BearerToken is the authentication token used for all requests to the control plane.
	BearerToken string

	// Region identifies the geographical region of this client instance.
	// This value is added as the X-Krane-Region header for proper request routing.
	Region string
}

// NewClient creates a new control plane client with the specified configuration.
//
// The returned client is configured with:
// - No timeout (infinite) for long-running operations
// - HTTP/2 transport with keepalive pings to prevent ALB idle timeout disconnections
// - Automatic authentication and metadata injection via interceptor
// - h2c (HTTP/2 cleartext) support for non-TLS URLs (local development)
//
// All outgoing requests will automatically include the Authorization bearer token
// and X-Krane-Region headers for proper routing and authentication.
func NewClient(cfg ClientConfig) ctrl.ClusterServiceClient {
	var transport http.RoundTripper

	// Use h2c (HTTP/2 cleartext) for non-TLS URLs, regular HTTP/2 for TLS
	if strings.HasPrefix(cfg.URL, "http://") {
		//nolint:exhaustruct
		transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				// For h2c, we dial without TLS
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
			ReadIdleTimeout: 10 * time.Second,
			PingTimeout:     5 * time.Second,
		}
	} else {
		//nolint:exhaustruct
		transport = &http2.Transport{
			ReadIdleTimeout: 10 * time.Second,
			PingTimeout:     5 * time.Second,
		}
	}

	return ctrl.NewConnectClusterServiceClient(ctrlv1connect.NewClusterServiceClient(
		&http.Client{
			Timeout:   0,
			Transport: transport,
		},
		cfg.URL,
		connect.WithInterceptors(connectInterceptor(cfg.Region, cfg.BearerToken)),
	))
}
