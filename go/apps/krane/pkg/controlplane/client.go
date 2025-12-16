package controlplane

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
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

	// Shard identifies the logical shard this client belongs to.
	// This value is added as the X-Krane-Shard header for proper request routing.
	Shard string
}

// NewClient creates a new control plane client with the specified configuration.
//
// The returned client is configured with:
// - No timeout (infinite) for long-running operations
// - HTTP transport with 1-hour idle connection timeout
// - Automatic authentication and metadata injection via interceptor
//
// All outgoing requests will automatically include the Authorization bearer token
// and X-Krane-Region/X-Krane-Shard headers for proper routing and authentication.
func NewClient(cfg ClientConfig) ctrlv1connect.ClusterServiceClient {
	return ctrlv1connect.NewClusterServiceClient(
		&http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				IdleConnTimeout: time.Hour,
			},
		},
		cfg.URL,
		connect.WithInterceptors(connectInterceptor(cfg.Region, cfg.Shard, cfg.BearerToken)),
	)
}
