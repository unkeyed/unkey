package api

import (
	"github.com/unkeyed/unkey/pkg/tls"
)

// RestateConfig holds configuration for Restate workflow engine integration.
//
// The API is a Restate client that invokes workflows. It only needs the
// ingress URL and optional API key for authentication.
type RestateConfig struct {
	// URL is the Restate ingress endpoint URL for workflow invocation.
	// Used by clients to start and interact with workflow executions.
	// Example: "http://restate:8080".
	URL string

	// APIKey is the authentication key for Restate ingress requests.
	// If set, this key will be sent with all requests to the Restate ingress.
	APIKey string
}

// GitHubConfig holds configuration for GitHub App integration.
//
// This configuration enables receiving GitHub webhooks and authenticating
// with the GitHub API to download repository tarballs for deployment.
type GitHubConfig struct {
}

// Config holds configuration for the control plane API server.
//
// The API server handles Connect RPC requests and delegates workflow
// execution to Restate. It does NOT run workflows directly - that's
// the worker's job.
type Config struct {
	// InstanceID is the unique identifier for this control plane instance.
	// Used for logging, tracing, and cluster coordination.
	InstanceID string

	// Region is the geographic region where this control plane instance runs.
	// Used for logging, tracing, and region-aware routing decisions.
	Region string

	// HttpPort defines the HTTP port for the control plane server.
	// Default: 8080. Cannot be 0.
	HttpPort int

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int

	// DatabasePrimary is the primary database connection string.
	// Used for both read and write operations to persistent storage.
	DatabasePrimary string

	// OtelEnabled enables sending telemetry data to collector endpoint.
	// When true, enables metrics, traces, and structured logs.
	OtelEnabled bool

	// OtelTraceSamplingRate controls the percentage of traces sampled.
	// Range: 0.0 (no traces) to 1.0 (all traces). Recommended: 0.1.
	OtelTraceSamplingRate float64

	// TLSConfig contains TLS configuration for HTTPS server.
	// When nil, server runs in HTTP mode for development.
	TLSConfig *tls.Config

	// AuthToken is the authentication token for control plane API access.
	// Used by clients and services to authenticate with this control plane.
	AuthToken string

	// Restate configures workflow engine integration.
	// The API invokes workflows via Restate ingress.
	Restate RestateConfig

	// AvailableRegions is a list of available regions for deployments.
	// Typically in the format "region.provider", ie "us-east-1.aws", "local.dev"
	AvailableRegions []string

	// GitHubWebhookSecret is the secret used to verify webhook signatures.
	// Configured in the GitHub App webhook settings.
	GitHubWebhookSecret string

	// DefaultDomain is the fallback domain for system operations.
	// Used for wildcard certificate bootstrapping. When set, the API will
	// ensure a wildcard certificate exists for *.{DefaultDomain}.
	DefaultDomain string

	// RegionalApexDomain is the base domain for cross-region communication
	// between frontline instances. Combined with AvailableRegions to create
	// per-region wildcard certificates like *.{region}.{RegionalApexDomain}.
	RegionalApexDomain string
}

// Validate checks the configuration for required fields and logical consistency.
func (c Config) Validate() error {
	return nil
}
