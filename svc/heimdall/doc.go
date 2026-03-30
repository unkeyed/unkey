// Package heimdall is the resource usage metering agent for Unkey's deployment
// platform. It runs as a Deployment and writes per-instance resource snapshots
// to ClickHouse every collection interval for billing and dashboards.
//
// Data sources:
//   - Metrics Server (metrics.k8s.io): CPU and memory usage per pod
//   - Hubble relay (gRPC): network egress with public/internal classification
//   - Pod informer: labels (workspace, app, deployment) and resource limits
//
// Each snapshot captures "at time T, instance X was using Y CPU, Z memory,
// and had limits A/B." No snapshot = the instance wasn't running.
package heimdall
