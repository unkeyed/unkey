package krane

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// S3Config holds S3 configuration for vault storage.
//
// This configuration is used when the vault service needs to store encrypted
// secrets in S3 for persistence and cross-node synchronization.
type S3Config struct {
	// URL is the S3 endpoint URL including protocol and region.
	// Examples: "https://s3.amazonaws.com" or "https://s3.us-west-2.amazonaws.com".
	URL string

	// Bucket is the S3 bucket name where encrypted vault data is stored.
	// The bucket must exist and be accessible with the provided credentials.
	Bucket string

	// AccessKeyID is the AWS access key ID for S3 authentication.
	// Must have permissions to read and write objects in the specified bucket.
	AccessKeyID string

	// AccessKeySecret is the AWS secret access key for S3 authentication.
	// Should be stored securely and rotated regularly.
	AccessKeySecret string
}

// Config holds configuration for the krane agent server.
//
// This configuration defines how the krane agent connects to Kubernetes,
// authenticates with container registries, handles secrets, and exposes metrics.
type Config struct {
	// InstanceID is the unique identifier for this krane agent instance.
	// Used for distributed tracing, logging correlation, and cluster coordination.
	// Must be unique across all running krane instances in the same cluster.
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

	// VaultMasterKeys are the encryption keys for vault operations.
	// Required for decrypting environment variable secrets. At least one key
	// must be provided when vault functionality is enabled.
	VaultMasterKeys []string

	// VaultS3 configures S3 storage for encrypted vault data.
	// Required when VaultMasterKeys are provided for persistent secrets storage.
	VaultS3 S3Config

	// RPCPort specifies the port for the gRPC server that exposes krane APIs.
	// The SchedulerService and optionally SecretsService are served on this port.
	// Must be a valid port number (1-65535).
	RPCPort int

	ControlPlaneURL    string
	ControlPlaneBearer string

	ClusterID string
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
