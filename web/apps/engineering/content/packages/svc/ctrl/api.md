---
title: api
description: "provides the control plane HTTP/2 server for Unkey's distributed infrastructure"
---

Package api provides the control plane HTTP/2 server for Unkey's distributed infrastructure.

The control plane coordinates deployment workflows, certificate management, and cluster operations across the Unkey platform. It exposes Connect RPC services over HTTP/2 and integrates with Restate for durable workflow execution.

### Architecture

The control plane sits at the center of Unkey's infrastructure. It coordinates sentinel instances that run customer workloads, Restate for durable workflow execution, build artifact storage, and ACME providers for automatic TLS certificates. Connect RPC services are exposed for core control plane operations, deployment workflows, ACME management, OpenAPI specs, and cluster coordination.

### Usage

Configure and start the control plane:

	cfg := api.Config{
	    InstanceID:      "ctrl-1",
	    HttpPort:        8080,
	    DatabasePrimary: "postgres://...",
	    Restate: api.RestateConfig{
	        URL:      "http://restate:8080",
	        AdminURL: "http://restate:9070",
	    },
	}
	if err := api.Run(ctx, cfg); err != nil {
	    log.Fatal(err)
	}

The server supports both HTTP/2 cleartext (h2c) for development and TLS for production. When \[Config.TLSConfig] is set, the server uses HTTPS; otherwise it uses h2c to allow HTTP/2 without TLS.

### Shutdown

The \[Run] function handles graceful shutdown when the provided context is cancelled. All active connections are drained, database connections closed, and telemetry flushed before the function returns.

## Constants

```go
const maxWebhookBodySize = 2 * 1024 * 1024 // 2 MB

```


## Functions

### func Run

```go
func Run(ctx context.Context, cfg Config) error
```

Run starts the control plane server with the provided configuration.

This function initializes all required services and starts the HTTP/2 Connect server. It performs these major initialization steps: 1. Validates configuration and initializes structured logging 2. Sets up OpenTelemetry if enabled 3. Initializes database and build storage 4. Creates Restate ingress client for invoking workflows 5. Starts HTTP/2 server with all Connect handlers

The server handles graceful shutdown when context is cancelled, properly closing all services and database connections.

Returns an error if configuration validation fails, service initialization fails, or during server startup. Context cancellation results in clean shutdown with nil error.


## Types

### type Config

```go
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

	// RegionalDomain is the base domain for cross-region communication
	// between frontline instances. Combined with AvailableRegions to create
	// per-region wildcard certificates like *.{region}.{RegionalDomain}.
	RegionalDomain string

	// CnameDomain is the base domain for custom domain CNAME targets.
	// Each custom domain gets a unique subdomain like "{random}.{CnameDomain}".
	// For production: "unkey-dns.com"
	// For local: "unkey.local"
	CnameDomain string
}
```

Config holds configuration for the control plane API server.

The API server handles Connect RPC requests and delegates workflow execution to Restate. It does NOT run workflows directly - that's the worker's job.

#### func (Config) Validate

```go
func (c Config) Validate() error
```

Validate checks the configuration for required fields and logical consistency.

### type GitHubWebhook

```go
type GitHubWebhook struct {
	db            db.Database
	restate       *restateingress.Client
	webhookSecret string
}
```

GitHubWebhook handles incoming GitHub App webhook events and triggers deployment workflows via Restate. It validates webhook signatures using the configured secret before processing any events.

#### func (GitHubWebhook) ServeHTTP

```go
func (s *GitHubWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

ServeHTTP validates the webhook signature and dispatches to event-specific handlers. Currently supports push events for triggering deployments. Unknown event types are acknowledged with 200 OK but not processed.

### type RestateConfig

```go
type RestateConfig struct {
	// URL is the Restate ingress endpoint URL for workflow invocation.
	// Used by clients to start and interact with workflow executions.
	// Example: "http://restate:8080".
	URL string

	// AdminURL is the Restate admin API endpoint for managing invocations.
	// Used for canceling invocations. Example: "http://restate:9070".
	AdminURL string

	// APIKey is the authentication key for Restate ingress requests.
	// If set, this key will be sent with all requests to the Restate ingress.
	APIKey string
}
```

RestateConfig holds configuration for Restate workflow engine integration.

The API is a Restate client that invokes workflows. It only needs the ingress URL and optional API key for authentication.

