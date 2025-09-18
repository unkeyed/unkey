// Package interceptors provides shared ConnectRPC interceptors for observability and tenant management.
//
// This package consolidates common interceptor functionality across all Unkey services:
//   - Metrics collection with OpenTelemetry
//   - Distributed tracing
//   - Structured logging
//   - Tenant authentication and context propagation
//
// Usage example:
//
//	import (
//	    "github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
//	    "go.opentelemetry.io/otel"
//	)
//
//	// Create interceptors with service-specific configuration
//	metricsInterceptor := interceptors.NewMetricsInterceptor(
//	    interceptors.WithServiceName("metald"),
//	    interceptors.WithMeter(otel.Meter("metald")),
//	    interceptors.WithActiveRequestsMetric(true),
//	)
//
//	loggingInterceptor := interceptors.NewLoggingInterceptor(
//	    interceptors.WithServiceName("metald"),
//	    interceptors.WithLogger(logger),
//	)
//
//	tenantInterceptor := interceptors.NewTenantAuthInterceptor(
//	    interceptors.WithServiceName("metald"),
//	    interceptors.WithTenantAuth(true, "/health.v1.HealthService/Check"),
//	)
//
//	// Apply interceptors to ConnectRPC handler
//	handler := connect.NewUnaryHandler(
//	    procedure,
//	    svc.Method,
//	    connect.WithInterceptors(
//	        tenantInterceptor,
//	        metricsInterceptor,
//	        loggingInterceptor,
//	    ),
//	)
package interceptors

import (
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
)

// NewDefaultInterceptors creates a standard set of interceptors with sensible defaults.
// This includes metrics, logging, and tenant authentication interceptors configured
// for the specified service.
//
// The interceptors are returned in the recommended order:
// 1. Tenant auth (extracts tenant context first)
// 2. Metrics (tracks all requests including auth failures)
// 3. Logging (logs final request/response details)
func NewDefaultInterceptors(serviceName string, opts ...Option) []connect.UnaryInterceptorFunc {
	// Merge service name with any provided options
	allOpts := append([]Option{WithServiceName(serviceName)}, opts...)

	// Create default meter if not provided
	defaultOpts := []Option{
		WithMeter(otel.Meter(serviceName)),
	}
	allOpts = append(defaultOpts, allOpts...)

	return []connect.UnaryInterceptorFunc{
		NewMetricsInterceptor(allOpts...),
		NewLoggingInterceptor(allOpts...),
	}
}

// AIDEV-NOTE: Interceptors are ordered for proper context propagation:
// 1. Tenant auth must run first to add tenant context
// 2. Metrics can then include tenant info in metrics
// 3. Logging runs last to capture the complete request lifecycle
