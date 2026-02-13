package routes

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/zen/validation"
)

// Services aggregates all dependencies required by API route handlers. It acts
// as a dependency injection container, allowing [Register] to wire up handlers
// without exposing individual dependencies throughout the codebase.
//
// This struct is constructed during server startup and passed to [Register].
// All fields except the optional configuration fields (ChproxyToken, Pprof*)
// must be non-nil for the API to function correctly.
type Services struct {
	// Database provides access to the primary MySQL database for persistence.
	Database db.Database

	// Keys handles API key authentication, verification, and authorization
	// checks for incoming requests.
	Keys keys.KeyService

	// ClickHouse stores analytics data including verification events,
	// rate limit events, and request metrics.
	ClickHouse clickhouse.ClickHouse

	// Validator performs request payload validation using struct tags.
	Validator *validation.Validator

	// Ratelimit provides distributed rate limiting across API requests.
	Ratelimit ratelimit.Service

	// Auditlogs records security-relevant events for compliance and debugging.
	Auditlogs auditlogs.AuditLogService

	// Caches holds various cache instances for performance optimization,
	// including API metadata, key data, and rate limit namespace caches.
	Caches caches.Caches

	// Vault provides encrypted storage for sensitive key material.
	Vault vault.Client

	// ChproxyToken authenticates requests to internal chproxy endpoints.
	// When empty, chproxy routes are not registered.
	ChproxyToken string

	// CtrlDeploymentClient communicates with the control plane for deployment
	// operations like creating and managing deployments.
	CtrlDeploymentClient ctrlv1connect.DeployServiceClient

	// PprofEnabled controls whether pprof profiling endpoints are registered.
	PprofEnabled bool

	// PprofUsername is the HTTP basic auth username for pprof endpoints.
	// Required when PprofEnabled is true.
	PprofUsername string

	// PprofPassword is the HTTP basic auth password for pprof endpoints.
	// Required when PprofEnabled is true.
	PprofPassword string

	// UsageLimiter tracks and enforces usage limits on API keys.
	UsageLimiter usagelimiter.Service

	// AnalyticsConnectionManager manages connections to analytics backends
	// for retrieving verification and usage data.
	AnalyticsConnectionManager analytics.ConnectionManager
}
