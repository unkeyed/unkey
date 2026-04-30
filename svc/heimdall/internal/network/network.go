package network

import (
	"errors"

	"k8s.io/apimachinery/pkg/types"
)

// Attach returns errors wrapping one of these sentinels so the collector
// can tag metrics by reason without string-matching. Each sentinel captures
// *why* the attach failed, which determines whether it's benign churn
// (pod already terminated, CNI netns gone) or a real problem (kernel
// refused the TCX attach).
var (
	// ErrSandboxNotFound means the pod's sandbox container is no longer
	// in containerd. Benign: happens routinely for Completed/Failed pods
	// still in the informer cache.
	ErrSandboxNotFound = errors.New("sandbox container not found")
	// ErrNetnsOpen means opening /var/run/netns/cni-<uuid> failed, usually
	// because the netns was torn down between the containerd lookup and
	// the open. Benign if transient.
	ErrNetnsOpen = errors.New("open CNI netns failed")
	// ErrVethLookup means the pod's CNI netns doesn't have a usable veth,
	// or netlink refused to list links. Indicates a CNI misconfiguration
	// or a timing race; investigate if sustained.
	ErrVethLookup = errors.New("veth lookup in pod netns failed")
	// ErrTCXAttach means the kernel rejected link.AttachTCX. Real problem:
	// either CAP_NET_ADMIN is missing, the kernel is too old (<6.6), or
	// TCX chain state is corrupted.
	ErrTCXAttach = errors.New("TCX attach failed")
	// ErrAttachQueueFull means the async attach worker pool can't accept
	// more work right now. Benign: the collector will re-request this UID
	// on the next tick. Guards against unbounded memory growth under a
	// pathological rollout storm.
	ErrAttachQueueFull = errors.New("attach queue full")
)

// Counters is the per-pod byte snapshot returned by Read. All four are
// monotonic accumulators (the tc program increments via
// __sync_fetch_and_add) so billing math is the same `max(counter) -
// min(counter)` shape as cpu_usage_usec.
type Counters struct {
	EgressPublicBytes   int64
	EgressPrivateBytes  int64
	IngressPublicBytes  int64
	IngressPrivateBytes int64
}

// zeroCounters is the all-zero Counters value returned by every fail-open
// path (read failed, attach failed, host-network pod, stub on macOS).
// Declared once here so callers don't need an exhaustruct-shaped literal
// at every fail site.
var zeroCounters = Counters{
	EgressPublicBytes:   0,
	EgressPrivateBytes:  0,
	IngressPublicBytes:  0,
	IngressPrivateBytes: 0,
}

// Reader is the per-process handle the collector talks to. The Linux
// implementation owns the loaded eBPF program plus the per-pod attach
// links; the non-Linux implementation is a no-op so the same code path
// compiles on macOS and in tests.
//
// All methods are safe for concurrent use.
type Reader interface {
	// Attach requests that the two tc programs be hooked onto the pod-side
	// eth0 inside the pod's net namespace. Non-blocking: the actual work
	// (containerd sandbox lookup + setns into the pod netns + two
	// link.AttachTCX calls, ~100-300ms total) happens on a bounded
	// background worker pool. Attach returns nil when the request was
	// accepted — success on the attach itself is reflected later by Read
	// starting to return non-zero counters.
	//
	// Why async: each Attach is dominated by a containerd gRPC and a few
	// netlink syscalls. Running them inline on the 5s collect tick serialises
	// the cost: 110 cold-start pods × 200ms = 22 seconds, which overruns
	// the tick budget and causes TryLock skips. A worker pool fans the
	// attach work out at bounded concurrency so collect() never blocks on
	// a single pod.
	//
	// Idempotent — re-requesting an already-attached or already-queued
	// pod is a no-op. Returns ErrAttachQueueFull only if the queue has
	// backed up beyond capacity (which means the next tick will re-request
	// anyway, no data lost).
	Attach(uid types.UID) error

	// Detach removes the tc attachments and drops the pod's BPF map
	// entry. Safe to call on a pod that was never attached.
	Detach(uid types.UID)

	// Read returns the current cumulative byte counters for the pod.
	// Returns zeros (no error) when the pod isn't attached or has not
	// yet seen any packets.
	Read(uid types.UID) (Counters, error)

	// MapEntries returns the current number of live entries in the BPF
	// per-veth counter map. The caller (collector) exposes this as a
	// Prometheus gauge so operators can alert before LRU eviction kicks
	// in (LRU silently drops traffic from the oldest entry when the map
	// is full). Returns 0 on the non-Linux stub.
	MapEntries() int

	// Close releases all attach links and frees the BPF map. Idempotent.
	Close() error
}

// NewReader is defined in the platform-specific files: network_linux.go
// for the real eBPF implementation, network_stub.go for the no-op stub
// on non-Linux. The build tag picks one at compile time.
