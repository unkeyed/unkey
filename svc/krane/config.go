package krane

import (
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
)

// Config holds configuration for the krane agent server.
//
// This configuration defines how the krane agent connects to Kubernetes,
// authenticates with container registries, handles secrets, and exposes metrics.
type Config struct {
	// InstanceID is the unique identifier for this krane agent instance.
	// Used for distributed tracing, logging correlation, and cluster coordination.
	// Must be unique across all running krane instances in the same cluster.
	InstanceID string

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and compliance requirements.
	// Must match the region identifier used by the underlying cloud platform
	// and control plane configuration.
	Region string

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

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int

	// VaultURL is the URL of the remote vault service (e.g., http://vault:8080).
	// Required for decrypting environment variable secrets.
	VaultURL string

	// VaultToken is the authentication token for the vault service.
	// Used to authenticate requests to the vault API.
	VaultToken string

	// RPCPort specifies the port for the gRPC server that exposes krane APIs.
	// The SchedulerService and optionally SecretsService are served on this port.
	// Must be a valid port number (1-65535).
	RPCPort int

	ControlPlaneURL    string
	ControlPlaneBearer string

	// OtelEnabled enables OpenTelemetry instrumentation for tracing and metrics.
	// When true, InitGrafana will be called to set up OTEL exporters.
	OtelEnabled bool

	// OtelTraceSamplingRate controls the sampling rate for traces (0.0 to 1.0).
	// Only used when OtelEnabled is true.
	OtelTraceSamplingRate float64

	// --- Logging sampler configuration ---

	// LogSampleRate is the baseline probability (0.0-1.0) of emitting log events.
	LogSampleRate float64

	// LogErrorSampleRate is the probability (0.0-1.0) of emitting events with errors.
	LogErrorSampleRate float64

	// LogSlowSampleRate is the probability (0.0-1.0) of emitting slow events.
	LogSlowSampleRate float64

	// LogSlowThreshold defines what duration qualifies as "slow" for sampling.
	LogSlowThreshold time.Duration
}

// Validate checks the configuration for required fields and logical consistency.
//
// Returns an error if required fields are missing or configuration values are invalid.
// This method should be called before starting the krane agent to ensure
// proper configuration and provide early feedback on configuration errors.
//
// Currently, this method always returns nil as validation is not implemented.
// Future implementations will validate required fields such as RPCPort,
// RegistryURL, and consistency between VaultMasterKeys and VaultS3 configuration.
func (c Config) Validate() error {
	return nil
}
