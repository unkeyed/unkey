//go:build linux

package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	containersapi "github.com/containerd/containerd/api/services/containers/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/heimdall/internal/containerdrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/apimachinery/pkg/types"
)

// These tests drive the attach state machine (Attach / Detach / Read /
// Reconcile plus the worker pool) with the containerd listing, the netns
// probe, and the TCX attach replaced by an in-memory fake node. They need
// no root, no BPF, and no containerd.
//
// The scenario they pin is a gVisor pod whose sandbox is recreated under
// the same pod UID (memory-limit kill takes the whole runsc sandbox down;
// kubelet builds a new one with a new netns). containerd keeps the stopped
// sandbox's container record, and its OCI spec still names the removed
// netns path, until kubelet GC runs RemovePodSandbox. The reader must end
// up attached to the new netns, not stuck on the stale one.

// fakeSandbox is one containerd sandbox container record.
type fakeSandbox struct {
	id    string
	netns string
}

// fakeLink records Close so tests can prove the old TCX links were
// released when the reader moved to a new netns.
type fakeLink struct {
	link.Link
	closed *atomic.Bool
}

func (l fakeLink) Close() error {
	l.closed.Store(true)
	return nil
}

// fakeCounterMap mirrors the pinned pod_counters map: keyed by netns
// cookie, one bpfCounters per pod.
type fakeCounterMap struct {
	mu      sync.Mutex
	entries map[uint64]bpfCounters
}

func (m *fakeCounterMap) Lookup(key, valueOut any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.entries[key.(uint64)]
	if !ok {
		return ebpf.ErrKeyNotExist
	}
	*valueOut.(*bpfCounters) = c
	return nil
}

func (m *fakeCounterMap) Delete(key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key.(uint64))
	return nil
}

func (m *fakeCounterMap) Close() error { return nil }

func (m *fakeCounterMap) set(cookie uint64, egressPublic uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[cookie] = bpfCounters{EgressPublic: egressPublic}
}

func (m *fakeCounterMap) has(cookie uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.entries[cookie]
	return ok
}

// fakeNode is the containerd + netns view the reader sees. Tests mutate it
// between ticks to replay a sandbox recreation.
type fakeNode struct {
	mu        sync.Mutex
	sandboxes []fakeSandbox   // what containerd lists for the pod UID, in list order
	liveNetns map[string]bool // netns paths whose bind mount still exists
	cookies   map[string]uint64
	closed    map[string]*atomic.Bool // netns path → whether its TCX links were closed
	gates     map[string]*attachGate
	counters  *fakeCounterMap
}

// attachGate makes attachEth0 for one netns block until the test releases
// it, so a test can pin a worker mid-TCX-work and interleave other calls.
type attachGate struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

// open lets the gated worker finish. Safe to call more than once, so tests
// can register it in t.Cleanup and a failed assertion cannot leave the
// worker parked (Close waits on the worker group).
func (g *attachGate) open() { g.once.Do(func() { close(g.release) }) }

func newFakeNode() *fakeNode {
	return &fakeNode{
		mu:        sync.Mutex{},
		sandboxes: nil,
		liveNetns: map[string]bool{},
		cookies:   map[string]uint64{},
		closed:    map[string]*atomic.Bool{},
		gates:     map[string]*attachGate{},
		counters:  &fakeCounterMap{mu: sync.Mutex{}, entries: map[uint64]bpfCounters{}},
	}
}

// gate arms a one-shot block on the next attachEth0 for sb's netns. Wait on
// entered to know the worker is inside; call open to let it finish.
func (n *fakeNode) gate(t *testing.T, sb fakeSandbox) *attachGate {
	t.Helper()
	n.mu.Lock()
	defer n.mu.Unlock()
	g := &attachGate{entered: make(chan struct{}), release: make(chan struct{}), once: sync.Once{}}
	n.gates[sb.netns] = g
	t.Cleanup(g.open)
	return g
}

// addSandbox registers a sandbox with a live netns and a unique cookie.
func (n *fakeNode) addSandbox(id string) fakeSandbox {
	n.mu.Lock()
	defer n.mu.Unlock()
	sb := fakeSandbox{id: id, netns: "/var/run/netns/cni-" + id}
	n.sandboxes = append(n.sandboxes, sb)
	n.liveNetns[sb.netns] = true
	n.cookies[sb.netns] = uint64(len(n.cookies) + 1)
	return sb
}

// stopSandbox replays StopPodSandbox: CNI DEL removes the netns file but
// the container record (and its spec) stays until RemovePodSandbox.
func (n *fakeNode) stopSandbox(sb fakeSandbox) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.liveNetns[sb.netns] = false
}

func (n *fakeNode) list(context.Context, *containersapi.ListContainersRequest, ...grpc.CallOption) (*containersapi.ListContainersResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]*containersapi.Container, 0, len(n.sandboxes))
	for _, sb := range n.sandboxes {
		spec, err := json.Marshal(&specs.Spec{
			Linux: &specs.Linux{
				Namespaces: []specs.LinuxNamespace{{Type: specs.NetworkNamespace, Path: sb.netns}},
			},
		})
		if err != nil {
			return nil, err
		}
		out = append(out, &containersapi.Container{
			ID:   sb.id,
			Spec: &anypb.Any{TypeUrl: "types.containerd.io/opencontainers/runtime-spec/1/Spec", Value: spec},
		})
	}
	return &containersapi.ListContainersResponse{Containers: out}, nil
}

func (n *fakeNode) netnsGone(path string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return !n.liveNetns[path]
}

func (n *fakeNode) attachEth0(path string) (*ebpf.Collection, link.Link, link.Link, uint64, error) {
	n.mu.Lock()
	live := n.liveNetns[path]
	g := n.gates[path]
	delete(n.gates, path)
	n.mu.Unlock()

	if g != nil {
		close(g.entered)
		<-g.release
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if !live {
		return nil, nil, nil, 0, fmt.Errorf("%w: open pod netns %s: %w", ErrNetnsOpen, path, os.ErrNotExist)
	}
	closed := new(atomic.Bool)
	n.closed[path] = closed
	return &ebpf.Collection{}, fakeLink{Link: nil, closed: closed}, fakeLink{Link: nil, closed: closed}, n.cookies[path], nil
}

func (n *fakeNode) cookie(sb fakeSandbox) uint64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.cookies[sb.netns]
}

func (n *fakeNode) linksClosed(sb fakeSandbox) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	c, ok := n.closed[sb.netns]
	return ok && c.Load()
}

func newTestReader(t *testing.T, node *fakeNode) *linuxReader {
	t.Helper()
	r := &linuxReader{
		podCounters:   node.counters,
		cd:            nil,
		listSandboxes: node.list,
		netnsGone:     node.netnsGone,
		attachEth0:    node.attachEth0,
		mu:            sync.Mutex{},
		attached:      make(map[types.UID]attachedPod),
		pending:       make(map[types.UID]uint64),
		nextGen:       0,
		attachQ:       make(chan attachRequest, attachQueueSize),
		workerWG:      sync.WaitGroup{},
		closed:        make(chan struct{}),
	}
	for i := 0; i < attachWorkers; i++ {
		r.workerWG.Go(r.runAttachWorker)
	}
	t.Cleanup(func() { require.NoError(t, r.Close()) })
	return r
}

// tick replays one collector tick for a live pod: reconcile against the
// informer set, request the attach, wait for the worker, then read.
func tick(t *testing.T, r *linuxReader, uid types.UID) (Counters, error) {
	t.Helper()
	r.Reconcile(map[types.UID]struct{}{uid: {}})
	require.NoError(t, r.Attach(uid))
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		_, pending := r.pending[uid]
		return !pending
	}, 2*time.Second, time.Millisecond)
	return r.Read(uid)
}

func attachedCookie(r *linuxReader, uid types.UID) (uint64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.attached[uid]
	return p.cookie, ok
}

// TestAttach_SandboxRecreatedAfterExit is the production sequence: the app
// container's TaskExit makes the collector Detach, the next tick re-attaches
// while only the stopped sandbox is listed, and the new sandbox appears one
// tick later, listed after the stale one.
func TestAttach_SandboxRecreatedAfterExit(t *testing.T) {
	node := newFakeNode()
	r := newTestReader(t, node)
	uid := types.UID("pod-1")

	old := node.addSandbox("sandbox-a")
	node.counters.set(node.cookie(old), 1000)
	c, err := tick(t, r, uid)
	require.NoError(t, err)
	require.EqualValues(t, 1000, c.EgressPublicBytes)

	// gVisor OOM: sandbox dies, kubelet stops it, the app container's
	// TaskExit reaches OnExit, which Detaches.
	node.stopSandbox(old)
	r.Detach(uid)

	// Next tick: kubelet has not created the new sandbox yet. Only the
	// stopped one is listed and its netns path is ENOENT.
	_, err = tick(t, r, uid)
	require.ErrorIs(t, err, ErrNotAttached)

	// New sandbox exists. containerd list order is arbitrary; the stale
	// record first is the order that used to lose.
	fresh := node.addSandbox("sandbox-b")
	node.counters.set(node.cookie(fresh), 42)

	c, err = tick(t, r, uid)
	require.NoError(t, err, "pod is live; the reader must re-attach to the new sandbox")
	require.EqualValues(t, 42, c.EgressPublicBytes)
	cookie, ok := attachedCookie(r, uid)
	require.True(t, ok)
	require.Equal(t, node.cookie(fresh), cookie)
}

// TestAttach_SandboxRecreatedWithoutExitEvent covers a dropped TaskExit and
// a coalesced informer update: nothing ever called Detach, so the attach
// record silently points at a dead netns. That is worse than an unattached
// checkpoint because the frozen counters look measured.
func TestAttach_SandboxRecreatedWithoutExitEvent(t *testing.T) {
	node := newFakeNode()
	r := newTestReader(t, node)
	uid := types.UID("pod-1")

	old := node.addSandbox("sandbox-a")
	node.counters.set(node.cookie(old), 1000)
	_, err := tick(t, r, uid)
	require.NoError(t, err)

	node.stopSandbox(old)
	fresh := node.addSandbox("sandbox-b")
	node.counters.set(node.cookie(fresh), 7)

	c, err := tick(t, r, uid)
	require.NoError(t, err)
	require.EqualValues(t, 7, c.EgressPublicBytes, "must read the new netns, not the frozen old counters")
	cookie, ok := attachedCookie(r, uid)
	require.True(t, ok)
	require.Equal(t, node.cookie(fresh), cookie)
	require.True(t, node.linksClosed(old), "old TCX links must be released")
	require.False(t, node.counters.has(node.cookie(old)), "old map entry must be dropped")
}

// TestAttach_CancelledRequestDoesNotCommitOverNewerOne pins the schedule
// where an attach for the old sandbox is mid-TCX-work when Detach cancels
// it and a new Attach for the same UID is queued. The old worker finishes
// first. It must not store its links (they point at the dead netns) and
// must not clear the new request's pending marker; the new worker then
// commits. Before request generations, the old worker saw the new pending
// entry as its own and did both.
func TestAttach_CancelledRequestDoesNotCommitOverNewerOne(t *testing.T) {
	node := newFakeNode()
	r := newTestReader(t, node)
	uid := types.UID("pod-1")

	old := node.addSandbox("sandbox-a")
	oldGate := node.gate(t, old)
	require.NoError(t, r.Attach(uid))
	<-oldGate.entered // worker 1 is inside attachEth0(old)

	// Sandbox dies; OnExit Detaches, which cancels worker 1's request.
	node.stopSandbox(old)
	r.Detach(uid)

	// New sandbox is up; the next tick queues a fresh request.
	fresh := node.addSandbox("sandbox-b")
	node.counters.set(node.cookie(fresh), 9)
	freshGate := node.gate(t, fresh)
	require.NoError(t, r.Attach(uid))
	<-freshGate.entered // worker 2 is inside attachEth0(fresh)

	// Worker 1 finishes first, with links built against the dead netns.
	oldGate.open()
	require.Eventually(t, func() bool { return node.linksClosed(old) }, 2*time.Second, time.Millisecond,
		"cancelled request must tear down its links")
	_, attached := attachedCookie(r, uid)
	require.False(t, attached, "cancelled request must not commit")
	r.mu.Lock()
	_, stillPending := r.pending[uid]
	r.mu.Unlock()
	require.True(t, stillPending, "cancelled request must not clear the newer request's marker")

	// Worker 2 finishes and commits.
	freshGate.open()
	require.Eventually(t, func() bool {
		cookie, ok := attachedCookie(r, uid)
		return ok && cookie == node.cookie(fresh)
	}, 2*time.Second, time.Millisecond)
	c, err := r.Read(uid)
	require.NoError(t, err)
	require.EqualValues(t, 9, c.EgressPublicBytes)
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return len(r.pending) == 0
	}, 2*time.Second, time.Millisecond)
}

// TestAttach_StableNetnsIsNotReattached guards the cost side: a healthy
// attached pod must not be torn down and rebuilt every tick.
func TestAttach_StableNetnsIsNotReattached(t *testing.T) {
	node := newFakeNode()
	r := newTestReader(t, node)
	uid := types.UID("pod-1")

	sb := node.addSandbox("sandbox-a")
	for i := 0; i < 3; i++ {
		_, err := tick(t, r, uid)
		require.NoError(t, err)
	}
	require.False(t, node.linksClosed(sb))
	require.Equal(t, 1, r.MapEntries())
}

func TestSandboxNetnsPath(t *testing.T) {
	node := newFakeNode()
	r := newTestReader(t, node)
	podUID := uid.New(uid.TestPrefix)
	server := grpc.NewServer()
	containersapi.RegisterContainersServer(server, &sandboxServer{node: node, podUID: podUID})
	socket := filepath.Join(t.TempDir(), "containerd.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		require.NoError(t, <-serveResult)
	})
	connection, err := containerdrpc.NewConnection(socket)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	r.listSandboxes = containersapi.NewContainersClient(connection).List
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	_, err = r.sandboxNetnsPath(ctx, podUID)
	require.ErrorIs(t, err, ErrSandboxNotFound, "no sandbox records at all")

	stale := node.addSandbox(uid.New(uid.TestPrefix))
	node.stopSandbox(stale)
	_, err = r.sandboxNetnsPath(ctx, podUID)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist, "only a stopped sandbox: netns gone, retry later")
	require.False(t, errors.Is(err, ErrSandboxNotFound))

	fresh := node.addSandbox(uid.New(uid.TestPrefix))
	path, err := r.sandboxNetnsPath(ctx, podUID)
	require.NoError(t, err)
	require.Equal(t, fresh.netns, path, "stale record listed first must not win")
}

type sandboxServer struct {
	containersapi.UnimplementedContainersServer
	node   *fakeNode
	podUID string
}

func (s *sandboxServer) List(ctx context.Context, req *containersapi.ListContainersRequest) (*containersapi.ListContainersResponse, error) {
	if values := metadata.ValueFromIncomingContext(ctx, "containerd-namespace"); len(values) != 1 || values[0] != "k8s.io" {
		return nil, status.Error(codes.InvalidArgument, "missing namespace")
	}
	expected := fmt.Sprintf(`labels."io.kubernetes.pod.uid"==%s,labels."io.cri-containerd.kind"==sandbox`, s.podUID)
	if len(req.Filters) != 1 || req.Filters[0] != expected {
		return nil, status.Error(codes.InvalidArgument, "incorrect sandbox filter")
	}
	return s.node.list(ctx, req)
}
