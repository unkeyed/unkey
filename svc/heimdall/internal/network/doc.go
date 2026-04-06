// Package network owns heimdall's per-pod egress/ingress byte accounting.
// The kernel-side program lives in bpf/network.bpf.c (compiled by bpf2go
// into Go bindings + an embedded .o; run `make generate-bpf` to regenerate)
// and attaches via TCX on each pod's host-side veth rather than at the pod
// cgroup - cgroup_skb is useless under gVisor because runsc traps customer
// syscalls in userspace and never hits the host socket layer.
//
// Counters are monotonic accumulators incremented via __sync_fetch_and_add,
// so billing math is the same max(counter) - min(counter) shape as the
// cpu_usage_usec flow. See docs/engineering/infra/metering/heimdall.mdx
// for the full design writeup including the TC_ACT_UNSPEC / TCX_NEXT trap
// that keeps DNS working for gVisor pods.
//
// Files:
//   - network.go          - Reader interface + Counters type (cross-platform)
//   - network_linux.go    - Linux implementation (TCX attach, map lookups)
//   - network_stub.go     - no-op implementation for macOS dev / tests
//   - sandbox_linux.go    - containerd sandbox → CNI netns path lookup
//   - veth_linux.go       - setns + netlink to find the host-side veth ifindex
//   - bpf/network.bpf.c   - the eBPF classifier
//   - bpf/network_helpers.h - hand-rolled subset of vmlinux.h + libbpf
package network
