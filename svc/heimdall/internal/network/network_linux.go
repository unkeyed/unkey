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
	podCounters counterMap         // shared pinned counter map, read by Read()
	cd          *containerd.Client // owned only for Close; lookups go through listSandboxes

	// The three effectful steps of an attach, plus the netns liveness probe,
	// are fields so the attach state machine can be driven in tests without
	// containerd, CAP_BPF, or a TCX-capable kernel. NewReader wires the real
	// implementations.
	listSandboxes func(ctx context.Context, filters ...string) ([]containerd.Container, error)
	netnsGone     func(path string) bool
	attachEth0    func(netnsPath string) (coll *ebpf.Collection, egress, ingress link.Link, cookie uint64, err error)

	mu       sync.Mutex
	attached map[types.UID]attachedPod // pod uid → per-pod links + collection + cookie
	pending  map[types.UID]uint64      // uids currently enqueued or being attached → request generation
	nextGen  uint64                    // generation handed to the next attach request

	// Async attach worker pool. Attach() enqueues; workers dequeue and
	// call attachSync. Keeps the 5s collect tick from serialising on
	// ~200ms-per-pod containerd/netns/TCX work during cold start or
	// rollout storms.
	attachQ  chan attachRequest
	workerWG sync.WaitGroup
	closed   chan struct{}
}

// attachRequest is one queued attach. gen is unique per request so a worker
// can tell whether the pending entry it sees at commit time is still its
// own: Detach cancels a request by clearing pending, and a later Attach for
// the same UID re-creates pending with a new gen. Without gen, the old
// worker would read the new request's entry as "still wanted", store links
// built against the netns that Detach was reacting to, and then clear the
// new request's marker on its way out.
type attachRequest struct {
	uid types.UID
	gen uint64
}

// attachedPod holds everything we allocated for one pod's attach: the
// two TCX links, the per-pod Collection (owns the two loaded programs),
// and the netns cookie we baked into POD_KEY (also the map key).
//
// netnsPath is kept so Attach can notice when the sandbox behind this
// record has been replaced. A pod UID outlives its sandbox under gVisor:
// a memory-limit kill takes down the whole runsc sandbox and kubelet
// builds a new one, with a new netns, for the same pod. The links here
// then sit on an eth0 that no longer exists and the map entry stops
// moving, while the informer still shows the pod Running.
type attachedPod struct {
	coll      *ebpf.Collection
	egress    link.Link
	ingress   link.Link
	cookie    uint64
	netnsPath string
}

// release closes the pod's TCX links, unloads its programs, and drops its
// counter map entry. Caller holds r.mu.
func (r *linuxReader) release(p attachedPod) {
	_ = p.ingress.Close()
	_ = p.egress.Close()
	p.coll.Close()
	_ = r.podCounters.Delete(p.cookie)
}

// counterMap is the subset of *ebpf.Map the reader uses. Read and Detach
// only need lookup and delete by cookie; naming the subset lets tests
// substitute an in-memory map where loading a real BPF map is impossible.
type counterMap interface {
	Lookup(key, valueOut any) error
	Delete(key any) error
	Close() error
}

// netnsFileGone reports whether the CNI netns bind mount at path has been
// removed. Only a definite ENOENT counts: any other stat failure (EACCES,
// a transient mount error) must not tear down a working attach.
func netnsFileGone(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
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

	pinOpts := ebpf.MapOptions{ //nolint:exhaustruct // LoadPinOptions optional
		PinPath: bpfPinDir,
	}
	r := &linuxReader{
		podCounters:   podCounters,
		cd:            cd,
		listSandboxes: cd.Containers,
		netnsGone:     netnsFileGone,
		// attachPodEth0 does everything netns-scoped in one locked-thread
		// block: enters the pod netns, finds eth0, reads the netns cookie,
		// clones the spec and bakes POD_KEY = cookie into rodata, loads the
		// per-pod program pair (sharing the pinned map via pinOpts),
		// attaches both TCX programs, and setns back.
		attachEth0: func(netnsPath string) (*ebpf.Collection, link.Link, link.Link, uint64, error) {
			return attachPodEth0(netnsPath, spec, pinOpts)
		},
		mu:       sync.Mutex{},
		attached: make(map[types.UID]attachedPod),
		pending:  make(map[types.UID]uint64),
		nextGen:  0,
		attachQ:  make(chan attachRequest, attachQueueSize),
		closed:   make(chan struct{}),
		workerWG: sync.WaitGroup{},
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
// The trailing version suffix (currently `/v2`) versions the pin. libbpf
// refuses to reuse a pinned map whose spec (max_entries, value struct,
// key type) differs from what the loader expects — the error reads
// "map spec is incompatible with existing map". That's the right default:
// silently resizing a map under a running program would corrupt counter
// state. So any change to `pod_counters` spec needs a fresh pin path:
// bump the suffix (`v2` → `v3`) when you change `max_entries`, the
// `counters` struct, or the key type. Old pins at the previous suffix
// become inert cruft on existing nodes (tiny, no runtime effect); fresh
// nodes pick up the new spec cleanly.
//
// History: v1 used a u32 ifindex key; v2 switched to a u64 netns cookie,
// triggering the KeySize-changed incompatibility on existing pins.
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
//  1. Attached and the netns is still there: return nil (no-op).
//  2. Already enqueued or being processed: return nil (no-op, dedup via pending set).
//  3. Not attached, or attached to a netns that has since been removed:
//     add to pending, push on the queue, return nil.
//
// Case 3's second half is the sandbox-recreation path. Stat'ing the
// recorded netns path each tick costs a few microseconds per pod and is
// the only signal that does not depend on a CRI exit event or an informer
// transition arriving; both are dropped in practice (containerd#3177,
// coalesced informer updates), and a missed one used to leave a dead
// attach record reporting frozen counters as measured.
//
// ErrAttachQueueFull is returned only when the queue has backed up beyond
// attachQueueSize. The caller's next tick will re-request the same UID.
func (r *linuxReader) Attach(uid types.UID) error {
	r.mu.Lock()
	if p, ok := r.attached[uid]; ok {
		if !r.netnsGone(p.netnsPath) {
			r.mu.Unlock()
			return nil
		}
		r.release(p)
		delete(r.attached, uid)
		metrics.NetworkReattaches.Inc()
		logger.Info("network attach: pod netns replaced, re-attaching",
			"pod_uid", string(uid), "netns_path", p.netnsPath)
	}
	if _, ok := r.pending[uid]; ok {
		r.mu.Unlock()
		return nil
	}
	r.nextGen++
	req := attachRequest{uid: uid, gen: r.nextGen}
	r.pending[uid] = req.gen
	r.mu.Unlock()

	select {
	case r.attachQ <- req:
		return nil
	case <-r.closed:
		r.mu.Lock()
		r.clearPending(req)
		r.mu.Unlock()
		return nil
	default:
		// Queue full. Roll back the pending entry so the collector's next
		// tick can try again. Treated as a transient error.
		r.mu.Lock()
		r.clearPending(req)
		r.mu.Unlock()
		return ErrAttachQueueFull
	}
}

// clearPending removes req's pending marker, but only if it is still req's
// own: Detach may have cancelled it and a newer request may own the entry
// now. Caller holds r.mu.
func (r *linuxReader) clearPending(req attachRequest) {
	if gen, ok := r.pending[req.uid]; ok && gen == req.gen {
		delete(r.pending, req.uid)
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
		case req := <-r.attachQ:
			err := r.attachSync(req)

			r.mu.Lock()
			r.clearPending(req)
			r.mu.Unlock()

			if err == nil {
				continue
			}

			// Fire-and-forget means the collector can't see this error;
			// surface it through the metric. Benign churn logs at Debug so
			// a pod that is between sandboxes for a few ticks (or a
			// Running pod whose sandbox keeps failing to come up) cannot
			// turn into a sustained warn stream; real kernel-side failures
			// log at Warn with the full wrap chain. Deeper per-step
			// diagnostics live in sandbox_linux.go and veth_linux.go.
			reason := attachFailureReason(err)
			metrics.NetworkAttachFailures.WithLabelValues(reason).Inc()
			if reason == "netns_gone" || reason == "sandbox_not_found" {
				logger.Debug("network attach: sandbox not ready, will retry next tick",
					"pod_uid", string(req.uid),
					"reason", reason,
				)
				continue
			}
			logger.Warn("network attach failed",
				"pod_uid", string(req.uid),
				"reason", reason,
				"error", err.Error(),
			)
		}
	}
}

// attachFailureReason maps an attachSync error onto the metric label.
// Lives here (not in the collector) now that the attach runs on a worker
// goroutine inside this package. Categories: benign churn vs real
// kernel-side failures.
//
// netns_gone is ENOENT on the CNI netns path: CNI DEL already ran for the
// sandbox we found. That is not proof the pod is gone. Under gVisor kubelet
// recreates the sandbox for the same pod UID after a memory-limit kill, and
// for a tick or two the only sandbox containerd lists is the stopped one.
// A permanent "terminated" mark on this reason once zeroed network billing
// for every recreated pod until the next heimdall restart.
func attachFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrNetnsOpen) && errors.Is(err, os.ErrNotExist):
		return "netns_gone"
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
func (r *linuxReader) attachSync(req attachRequest) error {
	uid := req.uid

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

	coll, egress, ingressLink, cookie, err := r.attachEth0(netnsPath)
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

	// Detach clears r.pending under the same lock; if our entry is gone,
	// or now belongs to a newer request for the same UID, Detach ran while
	// we were doing TCX work and the netns we attached to is the one it was
	// reacting to. Tear down rather than persist links to a vanished
	// container. The worker's post-attachSync clearPending is likewise
	// generation-checked so we never clear the newer request's marker.
	if gen, ok := r.pending[uid]; !ok || gen != req.gen {
		_ = egress.Close()
		_ = ingressLink.Close()
		coll.Close()
		r.mu.Unlock()
		return nil
	}

	r.attached[uid] = attachedPod{
		coll:      coll,
		egress:    egress,
		ingress:   ingressLink,
		cookie:    cookie,
		netnsPath: netnsPath,
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
//
// Also cancels any in-flight or queued attach for this UID. Without this,
// a Detach that arrives while attachSync is mid-TCX-work (~100-300ms)
// would no-op, then attachSync would store the freshly-built links into
// r.attached for a pod that's already gone — leaking TCX links and BPF
// map state until process restart. attachSync re-checks r.pending under
// the lock before storing and tears down its work if the entry was cleared.
func (r *linuxReader) Detach(uid types.UID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.pending, uid)

	p, ok := r.attached[uid]
	if !ok {
		return
	}

	r.release(p)
	delete(r.attached, uid)
}

// Read returns the current cumulative byte counters for the pod. One map
// lookup by POD_KEY (== pod netns cookie).
//
// Returns ErrNotAttached when this process has no attach record, which is not
// the same as zero bytes: the counter map is pinned so a restarted heimdall
// still faces a kernel counter holding the pod's month-to-date total. Reporting
// zero there made the following tick's delta the whole cumulative counter.
//
// A pod that is attached but has no map entry yet genuinely has seen no packets,
// so that path still returns zeros.
func (r *linuxReader) Read(uid types.UID) (Counters, error) {
	// Hold the lock across the lookup so the attach record and the map
	// entry are read as one state. Releasing it in between lets a
	// concurrent Detach or Reconcile delete the entry after we decided the
	// pod is attached, and the ErrKeyNotExist path below would then report
	// zero bytes as a measured value. The lookup is a single syscall on a
	// hash map; holding r.mu for it is cheaper than a mis-stamped
	// checkpoint.
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.attached[uid]
	if !ok {
		return zeroCounters, ErrNotAttached
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

// Reconcile evicts attached entries whose pod UID is absent from active
// (the current informer pod set). Must be called once per collection
// tick. It is the backstop for pods whose CRI exit event and informer
// status update were both missed: without it those entries accumulate
// across the process lifetime, leaking BPF program FDs.
//
// Reconcile does NOT touch pending — those entries self-clear when the
// worker finishes attachSync.
func (r *linuxReader) Reconcile(active map[types.UID]struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for uid, p := range r.attached {
		if _, ok := active[uid]; ok {
			continue
		}
		r.release(p)
		delete(r.attached, uid)
		metrics.NetworkReconcileEvictions.Inc()
	}
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

	r.pending = map[types.UID]uint64{}
	if r.cd != nil {
		_ = r.cd.Close()
	}

	// Close the shared map handle. The pinned inode stays on disk (under
	// bpfPinDir), so the next heimdall process reuses it — that's the point
	// of the pin. We're only releasing the userspace FD.
	return r.podCounters.Close()
}
