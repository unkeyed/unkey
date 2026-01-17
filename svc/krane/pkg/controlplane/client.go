package controlplane

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
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

	// ClusterID is the identifier of the cluster this client is associated with.
	// This value is added as the X-Krane-Cluster-Id header for proper request routing.
	ClusterID string
}

// NewClient creates a new control plane client with the specified configuration.
//
// The returned client is configured with:
// - No timeout (infinite) for long-running operations
// - HTTP/2 transport with keepalive pings to prevent ALB idle timeout disconnections
// - Automatic authentication and metadata injection via interceptor
//
// All outgoing requests will automatically include the Authorization bearer token
// and X-Krane-Region headers for proper routing and authentication.
func NewClient(cfg ClientConfig) ctrlv1connect.ClusterServiceClient {
	return ctrlv1connect.NewClusterServiceClient(
		&http.Client{
			Timeout: 0,
			Transport: &http2.Transport{
				// Send HTTP/2 PING frame if no frame is received within this duration.
				// This keeps the connection alive through AWS ALB's idle timeout (default 60s).
				ReadIdleTimeout: 10 * time.Second,
				// Time to wait for a PING response before considering the connection dead.
				PingTimeout: 5 * time.Second,
			},
		},
		cfg.URL,
		connect.WithInterceptors(connectInterceptor(cfg.Region, cfg.ClusterID, cfg.BearerToken)),
	)
}
