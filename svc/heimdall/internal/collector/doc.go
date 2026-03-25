// Package collector scrapes per-pod resource usage metrics from the local
// kubelet and writes them to ClickHouse via the shared [clickhouse.Bufferer].
//
// It reads CPU and memory from /metrics/resource, network egress from
// /stats/summary, and computes CPU rate from consecutive cumulative counter
// readings. Only krane-managed deployment pods are collected.
//
// The collector also supports immediate on-demand collection via [Collector.CollectOnce],
// used by the lifecycle tracker to grab a reading when a pod starts or stops.
package collector
