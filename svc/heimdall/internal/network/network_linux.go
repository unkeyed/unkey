//go:build linux

package network

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/internal/metrics"
	"k8s.io/apimachinery/pkg/types"
)

// linuxReader is the real eBPF-backed implementation of network.Reader.
// Sandbox netns resolution lives in sandbox_linux.go; pod-netns attach
// (setns + pod-side eth0 discovery + netns cookie read + per-pod program
// load with POD_KEY rewrite + TCX attach) lives in veth_linux.go. This
// file holds just the Reader interface methods and the async attach
// worker pool.
type linuxReader struct {
	spec        *ebpf.CollectionSpec // cached; cloned per pod to rewrite POD_KEY
	podCounters *ebpf.Map            // shared pinned counter map, read by Read()
	cd          *containerd.Client   // sandbox-container lookups for netns resolution

	mu       sync.Mutex
	attached map[types.UID]attachedPod // pod uid → per-pod links + collection + cookie
	pending  map[types.UID]struct{}    // uids currently enqueued or being attached

	// Async attach worker pool. Attach() enqueues; workers dequeue and
	// call attachSync. Keeps the 5s collect tick from serialising on
	// ~200ms-per-pod containerd/netns/TCX work during cold start or
	// rollout storms.
	attachQ  chan types.UID
	workerWG sync.WaitGroup
	closed   chan struct{}
}

// attachedPod holds everything we allocated for one pod's attach: the
// two TCX links, the per-pod Collection (owns the two loaded programs),
// and the netns cookie we baked into POD_KEY (also the map key).
type attachedPod struct {
	coll    *ebpf.Collection
	egress  link.Link
	ingress link.Link
	cookie  uint64
}

// attachWorkers bounds the number of goroutines doing containerd + netns +
// TCX work at once. 8 workers means the worst-case cold-start of 110 pods
// takes ~110 × 200ms / 8 ≈ 2.75s, small enough that the next 5s tick still
// sees every pod attached.
const attachWorkers = 8

// attachQueueSize bounds in-flight attach requests. Deliberately small:
// Attach is deduplicated via the `pending` set, so there's at most one
// entry per pod. 256 covers a rollout that lands hundreds of pods
// simultaneously; a burst larger than that overflows into ErrAttachQueueFull
// and the collector retries on the next tick.
const attachQueueSize = 256

// NewReader loads the eBPF program once per process and dials containerd
// at criSocket. Loads need RLIMIT_MEMLOCK lifted (rlimit.RemoveMemlock)
// and CAP_BPF on kernels >= 5.8 (or CAP_SYS_ADMIN on older). TCX attach
// additionally needs CAP_NET_ADMIN. The DaemonSet manifest grants all
// three. A failing containerd dial is fatal: without it, we can't resolve
// the CNI netns for any pod under gVisor (no IP on eth0 to match on) and
// a silently-broken collector would undercharge every pod.
//
// On non-Linux platforms this returns a no-op stubReader — see
// network_stub.go. The build tag switches the implementation at compile
// time so the caller sees one exported symbol regardless of platform.
func NewReader(criSocket string) (Reader, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("rlimit.RemoveMemlock: %w", err)
	}

	if criSocket == "" {
		return nil, errors.New("cri socket required for sandbox netns resolution")
	}

	cd, err := containerd.New(criSocket, containerd.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return nil, fmt.Errorf("dial containerd at %s: %w", criSocket, err)
	}

	// Pin the BPF map in bpffs so it survives heimdall pod restarts.
	// The C side declares `pod_counters` with LIBBPF_PIN_BY_NAME; the
	// loader needs a matching PinPath so the pinned map is reused across
	// restarts instead of recreated zero-filled. Without this, the per-pod
	// byte counters would appear to reset every time heimdall reloads and
	// billing queries that assume counter monotonicity would silently
	// undercount whatever traffic crossed the gap. The DaemonSet mounts
	// /sys/fs/bpf (hostPath) so this path lands on the host's bpffs.
	if err := os.MkdirAll(bpfPinDir, 0o700); err != nil {
		_ = cd.Close()
		return nil, fmt.Errorf("create bpf pin dir %s: %w", bpfPinDir, err)
	}

	// Parse the embedded spec once. Each pod attach clones this spec
	// (via spec.Copy()) to bake its unique POD_KEY into .rodata without
	// disturbing the cached version. The map spec inside also carries
	// LIBBPF_PIN_BY_NAME, so NewCollectionWithOptions below opens the
	// existing pin on restart rather than creating a fresh map.
	spec, err := loadBpf()
	if err != nil {
		_ = cd.Close()
		return nil, fmt.Errorf("loadBpf spec: %w", err)
	}

	// Load the shared counter map once up-front so Read/Detach have a
	// direct handle, and so restart-time pin reuse happens here rather
	// than per-pod (every per-pod load below will just re-open the same
	// pin). Strip the spec's programs for this load — we only want the
	// map and its pin semantics now.
	mapOnlySpec := &ebpf.CollectionSpec{ //nolint:exhaustruct // ByteOrder + Types default
		Maps:      spec.Maps,
		Variables: spec.Variables,
	}
	mapColl, err := ebpf.NewCollectionWithOptions(mapOnlySpec, ebpf.CollectionOptions{ //nolint:exhaustruct // Programs optional
		Maps: ebpf.MapOptions{ //nolint:exhaustruct // LoadPinOptions optional
			PinPath: bpfPinDir,
		},
	})
	if err != nil {
		_ = cd.Close()
		return nil, fmt.Errorf("open pinned pod_counters map: %w", err)
	}
	podCounters := mapColl.Maps["pod_counters"]
	if podCounters == nil {
		mapColl.Close()
		_ = cd.Close()
		return nil, fmt.Errorf("pod_counters map missing after load")
	}

	r := &linuxReader{
		spec:        spec,
		podCounters: podCounters,
		cd:          cd,
		mu:          sync.Mutex{},
		attached:    make(map[types.UID]attachedPod),
		pending:     make(map[types.UID]struct{}),
		attachQ:     make(chan types.UID, attachQueueSize),
		closed:      make(chan struct{}),
		workerWG:    sync.WaitGroup{},
	}

	for i := 0; i < attachWorkers; i++ {
		r.workerWG.Go(r.runAttachWorker)
	}

	return r, nil
}

// bpfPinDir is where we pin the counter map in bpffs. The DaemonSet mounts
// /sys/fs/bpf from the host; creating a sub-directory keeps our pins from
// colliding with any other BPF-using daemon on the node (Cilium pins under
// /sys/fs/bpf/tc/globals and similar).
//
// The `/v1` suffix versions the pin. libbpf refuses to reuse a pinned map
// whose spec (max_entries, value struct, key type) differs from what the
// loader expects — the error looks like
// "map spec is incompatible with existing map". That's the right default:
// silently resizing a map under a running program would corrupt counter
// state. But it means any future change to the `pod_counters` spec needs
// a fresh pin path, so we start on `/v1` now. Bump to `/v2` the day you
// change `max_entries`, the `counters` struct, or the key type; old `/v1`
// pins on existing nodes become inert cruft (tiny, no runtime effect),
// and fresh nodes pick up the new spec cleanly.
const bpfPinDir = "/sys/fs/bpf/heimdall/v2"

// PinDir returns the bpffs path heimdall pins its counter maps under.
// Stamped onto every checkpoint's attributes so old vs new pin generations
// can be told apart at query time without a node_id deploy-history join.
func PinDir() string { return bpfPinDir }

// Attach enqueues the pod for asynchronous attach by a worker. It does NOT
// block on the containerd/netns/TCX work. See the Reader interface doc for
// why this is async.
//
// Idempotent under three possible states:
//  1. Already attached: return nil (no-op).
//  2. Already enqueued or being processed: return nil (no-op, dedup via pending set).
//  3. Not seen before: add to pending, push on the queue, return nil.
//
// ErrAttachQueueFull is returned only when the queue has backed up beyond
// attachQueueSize. The caller's next tick will re-request the same UID.
func (r *linuxReader) Attach(uid types.UID) error {
	r.mu.Lock()
	if _, ok := r.attached[uid]; ok {
		r.mu.Unlock()
		return nil
	}
	if _, ok := r.pending[uid]; ok {
		r.mu.Unlock()
		return nil
	}
	r.pending[uid] = struct{}{}
	r.mu.Unlock()

	select {
	case r.attachQ <- uid:
		return nil
	case <-r.closed:
		r.mu.Lock()
		delete(r.pending, uid)
		r.mu.Unlock()
		return nil
	default:
		// Queue full. Roll back the pending entry so the collector's next
		// tick can try again. Treated as a transient error.
		r.mu.Lock()
		delete(r.pending, uid)
		r.mu.Unlock()
		return ErrAttachQueueFull
	}
}

// runAttachWorker pulls UIDs off the queue and performs the synchronous
// attach work. Exits when r.closed is signalled. We don't close r.attachQ
// on shutdown — closing it races with concurrent Attach calls that might
// still be in the middle of their send-select, and a send on a closed
// channel panics. Signalling via r.closed plus letting the channel get
// GC'd is simpler and race-free.
//
// Spawned via workerWG.Go so the Add(1)/defer Done() bookkeeping is
// implicit.
func (r *linuxReader) runAttachWorker() {
	for {
		select {
		case <-r.closed:
			return
		case uid := <-r.attachQ:
			err := r.attachSync(uid)

			r.mu.Lock()
			delete(r.pending, uid)
			r.mu.Unlock()

			if err != nil {
				// Fire-and-forget means the collector can't see this
				// error; surface it through the same metric it used to
				// observe before the move to async, plus a Warn log so
				// the full wrap chain (the part the metric label can't
				// capture) is visible without flipping to Debug level.
				// Deeper per-step diagnostics live at Debug in sandbox_linux.go
				// and veth_linux.go.
				reason := attachFailureReason(err)
				metrics.NetworkAttachFailures.WithLabelValues(reason).Inc()
				logger.Warn("network attach failed",
					"pod_uid", string(uid),
					"reason", reason,
					"error", err.Error(),
				)
			}
		}
	}
}

// attachFailureReason maps an attachSync error onto the metric label.
// Lives here (not in the collector) now that the attach runs on a worker
// goroutine inside this package. Categories: benign churn vs real
// kernel-side failures.
func attachFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrSandboxNotFound):
		return "sandbox_not_found"
	case errors.Is(err, ErrNetnsOpen):
		return "netns_open"
	case errors.Is(err, ErrVethLookup):
		return "veth_lookup"
	case errors.Is(err, ErrTCXAttach):
		return "tcx_attach"
	default:
		return "other"
	}
}

// attachSync does the real containerd + netns + TCX work for one pod. Blocks
// for ~100-300ms. Called only from attach workers.
func (r *linuxReader) attachSync(uid types.UID) error {
	// Short per-call timeout so a wedged containerd gRPC can't pin a
	// worker indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	netnsPath, err := r.sandboxNetnsPath(ctx, string(uid))
	if err != nil {
		// Tag with the right sentinel. sandboxNetnsPath returns
		// ErrSandboxNotFound directly when the containerd list is
		// empty; anything else is classified as netns_open for lack
		// of a clearer sub-category at this layer.
		if errors.Is(err, ErrSandboxNotFound) {
			return err
		}
		return fmt.Errorf("resolve sandbox netns: %w: %w", ErrNetnsOpen, err)
	}

	// attachPodEth0 does everything netns-scoped in one locked-thread
	// block: enters the pod netns, finds eth0, reads the netns cookie,
	// clones the spec and bakes POD_KEY = cookie into rodata, loads the
	// per-pod program pair (sharing the pinned map via pinOpts),
	// attaches both TCX programs, and setns back.
	coll, egress, ingressLink, cookie, err := attachPodEth0(netnsPath, r.spec, ebpf.MapOptions{ //nolint:exhaustruct // LoadPinOptions optional
		PinPath: bpfPinDir,
	})
	if err != nil {
		return fmt.Errorf("attach pod eth0 (netns=%s): %w", netnsPath, err)
	}

	r.mu.Lock()

	// Re-check under the lock — another worker may have raced us, or
	// Detach may have removed the pod between enqueue and now. On race,
	// tear down what we just built rather than leak a second per-pod
	// program pair.
	if _, ok := r.attached[uid]; ok {
		_ = egress.Close()
		_ = ingressLink.Close()
		coll.Close()
		r.mu.Unlock()
		return nil
	}

	r.attached[uid] = attachedPod{
		coll:    coll,
		egress:  egress,
		ingress: ingressLink,
		cookie:  cookie,
	}
	r.mu.Unlock()

	return nil
}

// Detach closes both tc links, unloads the per-pod programs, and drops
// the pod's BPF map entry. TCX links auto-detach on close, and the pod
// netns (with eth0) usually disappears at pod teardown anyway (taking
// the attachment with it), but eager cleanup keeps the map sparse,
// frees the LRU slot immediately, and releases the kernel-side program
// refcount so the verifier-checked objects GC out.
func (r *linuxReader) Detach(uid types.UID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.attached[uid]
	if !ok {
		return
	}

	_ = p.ingress.Close()
	_ = p.egress.Close()
	p.coll.Close()
	_ = r.podCounters.Delete(p.cookie)
	delete(r.attached, uid)
}

// Read returns the current cumulative byte counters for the pod. One map
// lookup by POD_KEY (== pod netns cookie).
//
// Returns zeros (no error) when the pod isn't attached or hasn't seen any
// packets yet - indistinguishable from all-zero.
func (r *linuxReader) Read(uid types.UID) (Counters, error) {
	r.mu.Lock()
	p, ok := r.attached[uid]
	r.mu.Unlock()
	if !ok {
		return zeroCounters, nil
	}

	var c bpfCounters
	if err := r.podCounters.Lookup(p.cookie, &c); err != nil {
		if errors.Is(err, ebpf.ErrKeyNotExist) {
			return zeroCounters, nil
		}
		return zeroCounters, fmt.Errorf("map lookup cookie=%d: %w", p.cookie, err)
	}

	return Counters{
		EgressPublicBytes:   int64(c.EgressPublic),
		EgressPrivateBytes:  int64(c.EgressPrivate),
		IngressPublicBytes:  int64(c.IngressPublic),
		IngressPrivateBytes: int64(c.IngressPrivate),
	}, nil
}

// MapEntries returns the number of attached pods, which equals the number
// of live entries we've inserted into the BPF counter map. Using the Go
// side bookkeeping avoids iterating the map on every tick; drift is
// possible only if the kernel's LRU evicts an entry while we still think
// we have it attached (the next Read returns zeros, which is safe).
func (r *linuxReader) MapEntries() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.attached)
}

func (r *linuxReader) Close() error {
	// Signal Attach callers and workers (both select on r.closed). Workers
	// exit their for-select; any UIDs still in the queue are dropped (we
	// are shutting down, and the collector will re-request them if needed
	// on the next lifecycle). We deliberately do NOT close r.attachQ —
	// closing it races with in-flight sends from Attach and would panic.
	close(r.closed)
	r.workerWG.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()

	for uid, p := range r.attached {
		_ = p.ingress.Close()
		_ = p.egress.Close()
		p.coll.Close()
		delete(r.attached, uid)
	}

	r.pending = map[types.UID]struct{}{}
	_ = r.cd.Close()

	// Close the shared map handle. The pinned inode stays on disk (under
	// /sys/fs/bpf/heimdall/v1), so the next heimdall process reuses it —
	// that's the point of the pin. We're only releasing the userspace FD.
	return r.podCounters.Close()
}
