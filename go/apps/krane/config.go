package krane

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
)

// S3Config holds S3 configuration for vault storage
type S3Config struct {
	URL             string
	Bucket          string
	AccessKeyID     string
	AccessKeySecret string
}

// Backend represents the container orchestration backend type.
type Backend string

const (
	// Docker backend uses Docker Engine API for container management.
	// Suitable for local development and single-node deployments.
	Docker Backend = "docker"

	// Kubernetes backend uses Kubernetes API for container orchestration.
	// Designed for production multi-node cluster deployments.
	Kubernetes Backend = "kubernetes"
)

// Config holds krane server configuration for Docker or Kubernetes backends.
type Config struct {
	// InstanceID is the unique identifier for this instance of the API server.
	// Used for distributed tracing, logging correlation, and cluster coordination.
	// Should be unique across all running krane instances.
	InstanceID string

	// Platform identifies the cloud platform where the node is running.
	// Examples: "aws", "gcp", "azure", "hetzner", "bare-metal"
	// Used for observability tagging and platform-specific optimizations.
	Platform string

	// Image specifies the container image identifier including repository and tag.
	// This field may be deprecated in future versions as images should be
	// specified per-deployment rather than globally.
	Image string

	// HttpPort defines the HTTP port for the API server to listen on.
	// Default is 7070. The server uses HTTP/2 with h2c for Connect protocol.
	// Must be available and not conflicting with other services on the host.
	HttpPort int

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and compliance requirements.
	// Should match the region identifier used by the underlying cloud platform.
	Region string

	// Backend specifies the container orchestration system to use.
	// Must be either Docker or Kubernetes. Determines which backend
	// implementation is instantiated and which configuration fields are required.
	Backend Backend

	// DockerSocketPath specifies the Docker daemon socket path.
	// Required when Backend is Docker. Common values:
	//   - "/var/run/docker.sock" (Linux)
	//   - "/var/run/docker.sock" (macOS with Docker Desktop)
	// The path must be accessible by the krane process with appropriate permissions.
	DockerSocketPath string

	// RegistryURL is the URL of the container registry for pulling images.
	// Example: "registry.depot.dev"
	RegistryURL string

	// RegistryUsername is the username for authenticating with the container registry.
	// Example: "x-token", "depot", or any registry-specific username.
	RegistryUsername string

	// RegistryPassword is the password/token for authenticating with the container registry.
	// Should be stored securely (e.g., environment variable).
	RegistryPassword string

	// OtelEnabled controls whether OpenTelemetry observability data is collected
	// and sent to configured collectors. When enabled, the service exports
	// distributed traces, metrics, and structured logs.
	OtelEnabled bool

	// OtelTraceSamplingRate determines the fraction of traces to sample.
	// Range: 0.0 (no sampling) to 1.0 (sample all traces).
	// Recommended values: 0.1 for high-traffic production, 1.0 for development.
	OtelTraceSamplingRate float64

	// DeploymentEvictionTTL specifies the duration after which idle deployments
	// are automatically removed from the system. This prevents resource accumulation
	// in development and testing environments.
	//
	// Set to 0 or negative values to disable automatic eviction.
	// Recommended values:
	//   - Development: 1-4 hours
	//   - Staging: 24 hours
	//   - Production: disabled (0)
	DeploymentEvictionTTL time.Duration

	// Clock provides time operations for testing and time zone handling.
	// Use clock.RealClock{} for production, mock clocks for testing.
	Clock clock.Clock

	// VaultMasterKeys are the encryption keys for vault operations.
	// Required for decrypting environment variable secrets.
	VaultMasterKeys []string

	// VaultS3 configures S3 storage for encrypted vault data.
	VaultS3 S3Config

	// WorkloadNamespace is the Kubernetes namespace where customer workloads run.
	// Used by the K8s token validator to verify pod identity.
	// Only used when Backend is Kubernetes.
	WorkloadNamespace string
}

func (c Config) Validate() error {
	if c.Backend == Docker && c.DockerSocketPath == "" {
		return fmt.Errorf("--docker-socket is required when backend is docker")
	}

	return nil
}
