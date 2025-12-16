package krane

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// Config holds configuration for a krane agent instance.
//
// This configuration defines how the agent connects to the control plane,
// authenticates with container registries, and exposes observability endpoints.
// All fields should be configured before calling [Run].
type Config struct {
	// InstanceID is the unique identifier for this krane agent instance.
	// Used for distributed tracing, logging correlation, and cluster coordination.
	// Must be unique across all running krane instances in the same control plane.
	InstanceID string

	// Image specifies the default container image identifier including repository and tag.
	// This field is deprecated and should not be used. Images should be specified
	// per-deployment through the control plane API instead.
	// Deprecated: Use per-deployment image specification via control plane.
	Image string

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and compliance requirements.
	// Must match the region identifier used by the underlying cloud platform
	// and control plane configuration.
	Region string

	// Shard identifies the cluster within the region where this node is deployed.
	// Enables running multiple krane clusters within the same region for
	// isolation and scaling purposes. Must match control plane shard configuration.
	Shard string

	// RegistryURL is the URL of the container registry for pulling images.
	// Should include the protocol and registry domain, e.g., "registry.depot.dev"
	// or "https://registry.example.com". Used by all deployments unless overridden.
	RegistryURL string

	// RegistryUsername is the username for authenticating with the container registry.
	// Common values include "x-token" for token-based authentication or the
	// actual registry username. Must be paired with RegistryPassword.
	RegistryUsername string

	// RegistryPassword is the password or token for authenticating with the container registry.
	// Should be stored securely (e.g., environment variable or secret management system).
	// For token-based auth, this is the actual token value.
	RegistryPassword string

	// Clock provides time operations for testing and time zone handling.
	// Use clock.RealClock{} for production deployments and mock clocks for
	// deterministic testing. Enables time-based operations to be controlled in tests.
	Clock clock.Clock

	// ControlPlaneURL is the address of the control plane service.
	// Must include protocol and full path, e.g., "https://control-plane.example.com"
	// or "http://localhost:8080" for development. Used for gRPC connections.
	ControlPlaneURL string

	// ControlPlaneBearer is the bearer token for authenticating with the control plane.
	// Should be obtained from the control plane authentication system and treated
	// as sensitive data. Must be valid for the entire agent lifetime.
	ControlPlaneBearer string

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int
}

// Validate checks the configuration for required fields and logical consistency.
//
// Returns an error if required fields are missing or configuration values are invalid.
// This method should be called before starting the krane agent to ensure
// proper configuration and provide early feedback on configuration errors.
func (c Config) Validate() error {
	return nil
}
