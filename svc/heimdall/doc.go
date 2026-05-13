// Package heimdall is the per-node resource metering agent for Unkey's
// deployment platform. It runs as a DaemonSet and writes per-container raw
// counter checkpoints to ClickHouse every collection interval; billing math
// (max(counter)-min(counter), memory pair-integration) is deferred to query
// time.
//
// Data sources:
//   - cgroup v2 files (/sys/fs/cgroup): cpu.stat usage_usec, memory.current,
//     memory.stat inactive_file
//   - TC/eBPF on the pod's host-side veth: ingress/egress public/private
//     byte counters, attached lazily on first observation of each pod
//   - containerd CRI events + pod informer: lifecycle (start/stop) plus
//     pod metadata (labels, QoS, resource limits)
//
// Each checkpoint captures "at time T, container_uid=X had counter values
// CPU=Y, MEM=Z, ..." so a missing checkpoint = the container wasn't running.
package heimdall
