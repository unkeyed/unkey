// Package lifecycle watches Kubernetes pod events via informers and emits
// deployment lifecycle events (started, stopped, stopping) with millisecond-
// precise timestamps to ClickHouse via the shared [clickhouse.Bufferer].
//
// These events define the exact billing window boundaries — when a deployment
// was running and with what resource allocation — enabling ms-fair billing
// for both active (usage-based) and allocated (reservation-based) models.
package lifecycle
