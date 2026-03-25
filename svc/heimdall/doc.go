// Package heimdall is the resource usage metering agent for Unkey's deployment
// platform. It runs as a DaemonSet on each node hosting customer workloads and
// collects per-deployment CPU, memory, and network egress metrics for billing.
//
// heimdall has two data streams:
//
//   - Resource usage samples: scraped from the kubelet every collection interval,
//     providing actual CPU/memory consumption and network egress per pod.
//
//   - Lifecycle events: emitted via Kubernetes pod informers when deployments
//     start, stop, or scale, with millisecond-precise timestamps for billing
//     window boundaries.
//
// Both streams are written to ClickHouse via batch processors. Materialized
// views aggregate the raw data into per-minute, per-hour, per-day, and
// per-month tables for dashboard queries.
//
// # Usage
//
//	cfg, err := config.Load[heimdall.Config]("heimdall.toml")
//	if err != nil { ... }
//
//	return heimdall.Run(ctx, cfg)
package heimdall
