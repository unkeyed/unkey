// Package metrics provides Prometheus metric collectors for monitoring Unkey services.
//
// This package centralizes metric definitions to ensure consistent naming and labeling
// across all services. Metrics are registered automatically via [promauto] with the
// "unkey" namespace, making them available for scraping without manual registration.
//
// The package intentionally keeps metric definitions simple and focused. Each metric
// serves a specific observability purpose and includes labels that enable meaningful
// filtering and aggregation in dashboards and alerts.
//
// # Available Metrics
//
// [PanicsTotal] tracks recovered panics from HTTP handlers and background tasks.
// Use it to monitor application stability and identify code paths that need attention.
//
// # Usage
//
// Increment the panic counter when recovering from a panic:
//
//	metrics.PanicsTotal.WithLabelValues("handlerName", "/api/path").Inc()
//
// For background tasks, use a descriptive caller name and a synthetic path:
//
//	metrics.PanicsTotal.WithLabelValues("repeat.Every", "background").Inc()
package metrics
