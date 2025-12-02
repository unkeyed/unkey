package krane

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// Config holds krane server configuration for Docker or Kubernetes backends.
type Config struct {
	// InstanceID is the unique identifier for this instance of the API server.
	// Used for distributed tracing, logging correlation, and cluster coordination.
	// Should be unique across all running krane instances.
	InstanceID string

	// Image specifies the container image identifier including repository and tag.
	// This field may be deprecated in future versions as images should be
	// specified per-deployment rather than globally.
	Image string

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and compliance requirements.
	// Should match the region identifier used by the underlying cloud platform.
	Region string

	// Shard identifies the cluster within the region where this node is deployed.
	// Used to allow us to run multiple clusters within a region.
	Shard string

	// RegistryURL is the URL of the container registry for pulling images.
	// Example: "registry.depot.dev"
	RegistryURL string

	// RegistryUsername is the username for authenticating with the container registry.
	// Example: "x-token", "depot", or any registry-specific username.
	RegistryUsername string

	// RegistryPassword is the password/token for authenticating with the container registry.
	// Should be stored securely (e.g., environment variable).
	RegistryPassword string

	// Clock provides time operations for testing and time zone handling.
	// Use clock.RealClock{} for production, mock clocks for testing.
	Clock clock.Clock

	// ControlPlaneURL is the address of the control plane service.
	// Example: "http://localhost:8080"
	ControlPlaneURL    string
	ControlPlaneBearer string

	PrometheusPort int
}

func (c Config) Validate() error {

	return nil
}
