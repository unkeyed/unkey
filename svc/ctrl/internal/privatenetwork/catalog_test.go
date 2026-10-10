package privatenetwork

import (
	"context"
	"errors"
	"maps"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/proto"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/query"
)

var metricsRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(metricsRegistry)
	os.Exit(m.Run())
}

func builds(t *testing.T) float64 {
	t.Helper()
	families, err := metricsRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "unkey_control_private_network_catalog_builds_total" {
			return family.GetMetric()[0].GetCounter().GetValue()
		}
	}
	return 0
}

type fakeStore struct {
	mu         sync.Mutex
	active     map[string]caller
	apps       map[string]string
	topologies int
	reads      [][]string
	fail       error
	block      chan struct{}
}

func newFakeStore(active map[string]caller) *fakeStore {
	return &fakeStore{mu: sync.Mutex{}, active: maps.Clone(active), apps: map[string]string{}, topologies: 0, reads: nil, fail: nil, block: nil}
}

func (s *fakeStore) activeCallers(_ context.Context, _ string, deploymentIDs []string) (map[string]caller, error) {
	s.mu.Lock()
	block := s.block
	s.mu.Unlock()
	if block != nil {
		<-block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads = append(s.reads, slices.Clone(deploymentIDs))
	if s.fail != nil {
		return nil, s.fail
	}
	result := map[string]caller{}
	for _, id := range deploymentIDs {
		entry, ok := s.active[id]
		if !ok {
			continue
		}
		connections := make([]*ctrlv1.PrivateNetworkConnection, 0, len(entry.connections))
		for _, connection := range entry.connections {
			connection = proto.CloneOf(connection)
			connection.CallerDeploymentId = id
			connections = append(connections, connection)
		}
		entry.connections = connections
		result[id] = entry
	}
	return result, nil
}

func (s *fakeStore) deploymentApps(_ context.Context, deploymentIDs []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var apps []string
	for _, id := range deploymentIDs {
		if app, ok := s.apps[id]; ok {
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func (s *fakeStore) topology(context.Context, string) (*ctrlv1.PrivateNetworkTopology, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.topologies++
	return &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: nil}, nil
}

func (s *fakeStore) set(id string, entry caller) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[id] = entry
}

func (s *fakeStore) unset(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, id)
}

func (s *fakeStore) failWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fail = err
}

func (s *fakeStore) blockReads(block chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.block = block
}

type storeWork struct {
	callerReads   int
	topologyReads int
	lastCallers   []string
}

func (s *fakeStore) work() storeWork {
	s.mu.Lock()
	defer s.mu.Unlock()
	work := storeWork{callerReads: len(s.reads), topologyReads: s.topologies, lastCallers: nil}
	if len(s.reads) > 0 {
		work.lastCallers = s.reads[len(s.reads)-1]
	}
	return work
}

type streamStep struct {
	event cdc.Event
	end   error
	done  chan error
}

type fakeStream struct {
	starts chan []byte
	steps  chan streamStep
}

func newFakeStream() *fakeStream {
	return &fakeStream{starts: make(chan []byte, 8), steps: make(chan streamStep)}
}

func (s *fakeStream) watch(ctx context.Context, token []byte, send func(cdc.Event) error) error {
	s.starts <- slices.Clone(token)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case step := <-s.steps:
			if step.end != nil {
				step.done <- nil
				return step.end
			}
			err := send(step.event)
			step.done <- err
			if err != nil {
				return err
			}
		}
	}
}

func (s *fakeStream) send(event cdc.Event) chan error {
	done := make(chan error, 1)
	s.steps <- streamStep{event: event, end: nil, done: done}
	return done
}

func (s *fakeStream) deliver(t *testing.T, event cdc.Event) error {
	t.Helper()
	return <-s.send(event)
}

func (s *fakeStream) end(t *testing.T, err error) {
	t.Helper()
	done := make(chan error, 1)
	s.steps <- streamStep{event: cdc.Event{}, end: err, done: done}
	<-done
}

func (s *fakeStream) started(t *testing.T) []byte {
	t.Helper()
	select {
	case token := <-s.starts:
		return token
	case <-time.After(5 * time.Second):
		t.Fatal("catalog did not start a watch")
		return nil
	}
}

func (h catalogHarness) restarted(t *testing.T) []byte {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		h.clock.Tick(minBackoff)
		select {
		case token := <-h.stream.starts:
			return token
		case <-time.After(10 * time.Millisecond):
		case <-deadline:
			t.Fatal("catalog did not restart its watch")
			return nil
		}
	}
}

func rowEvent(table string, before, after []string) cdc.Event {
	image := func(values []string) *query.Row {
		if values == nil {
			return nil
		}
		row := &query.Row{Lengths: nil, Values: nil}
		for _, value := range values {
			row.Lengths = append(row.Lengths, int64(len(value)))
			row.Values = append(row.Values, value...)
		}
		return row
	}
	return cdc.Event{Change: &binlog.VEvent{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{
		TableName:  "unkey." + table,
		RowChanges: []*binlog.RowChange{{Before: image(before), After: image(after)}},
	}}}
}

func (h catalogHarness) checkpoint(token string, age time.Duration) cdc.Event {
	return cdc.Event{ResumeToken: []byte(token), CommitTime: h.clock.Now().Add(-age).Truncate(time.Second)}
}

func checkpointWithoutCommitTime(token string) cdc.Event {
	return cdc.Event{ResumeToken: []byte(token), CommitTime: time.Time{}}
}

var (
	copyCompleted = cdc.Event{CopyCompleted: true}
	heartbeat     = cdc.Event{Heartbeat: true}
)

func callerOf(workspace, app string, targetApps ...string) caller {
	entry := caller{workspaceID: workspace, appID: app, replica: nil, connections: nil}
	for _, target := range targetApps {
		entry.connections = append(entry.connections, &ctrlv1.PrivateNetworkConnection{TargetAppId: target, ConnectionId: uid.New(uid.ConnectionPrefix)})
	}
	return entry
}

type catalogHarness struct {
	catalog *catalog
	store   *fakeStore
	stream  *fakeStream
	clock   *clock.TestClock
}

func runCatalog(t *testing.T, active map[string]caller) catalogHarness {
	t.Helper()
	h := catalogHarness{
		catalog: nil,
		store:   newFakeStore(active),
		stream:  newFakeStream(),
		clock:   clock.NewTestClock(time.Unix(1_700_000_000, 0)),
	}
	h.catalog = newCatalog("aws", h.store, h.stream.watch, h.clock)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.catalog.run(ctx)
	}()
	t.Cleanup(func() { cancel(); <-done })
	require.Nil(t, h.stream.started(t))
	return h
}

func startCatalog(t *testing.T, active map[string]caller) catalogHarness {
	t.Helper()
	h := runCatalog(t, active)
	h.copy(t, active)
	require.NoError(t, h.stream.deliver(t, h.checkpoint("t0", time.Second)))
	return h
}

func (h catalogHarness) copy(t *testing.T, active map[string]caller) {
	t.Helper()
	for _, id := range slices.Sorted(maps.Keys(active)) {
		require.NoError(t, h.stream.deliver(t, rowEvent("deployments", nil, []string{id, active[id].appID})))
	}
	require.NoError(t, h.stream.deliver(t, copyCompleted))
}

func (h catalogHarness) snapshot(t *testing.T) Snapshot {
	t.Helper()
	snapshot, err := h.catalog.served.get()
	require.NoError(t, err)
	return snapshot
}

func callersIn(snapshot Snapshot) []string {
	callers := set{}
	for _, connection := range snapshot.Connections {
		callers.add(connection.GetCallerDeploymentId())
	}
	return slices.Sorted(maps.Keys(callers))
}

// TestCatalogPublishesOnlyAfterFinalCopyCompletion guarantees that Krane never
// receives a catalog built from part of the initial copy, because omission
// authorizes deletion.
func TestCatalogPublishesOnlyAfterFinalCopyCompletion(t *testing.T) {
	callerID, appID := uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix)
	h := runCatalog(t, map[string]caller{callerID: callerOf(uid.New(uid.WorkspacePrefix), appID, uid.New(uid.AppPrefix))})
	waitCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", nil, []string{callerID, appID})))
	require.NoError(t, h.stream.deliver(t, checkpointWithoutCommitTime("copying")))
	require.NoError(t, h.stream.deliver(t, heartbeat))
	work := h.store.work()
	require.Zero(t, work.callerReads+work.topologyReads, "checkpoints and heartbeats during the copy must not build a catalog")
	_, err := h.catalog.served.wait(waitCtx)
	require.ErrorIs(t, err, ErrUnavailable)

	require.NoError(t, h.stream.deliver(t, copyCompleted))
	work = h.store.work()
	require.Equal(t, 1, work.callerReads)
	require.Equal(t, []string{callerID}, work.lastCallers)
	snapshot, err := h.catalog.served.wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{callerID}, callersIn(snapshot))
	require.False(t, snapshot.Certified, "a copy is not certified before it applies a checkpoint")

	h.stream.end(t, errors.New("stream lost"))
	require.Nil(t, h.restarted(t), "a checkpoint seen during the copy is not a resume position")
}

// TestCatalogUnchangedStateDoesNoWork guarantees that polling, an idle stream,
// checkpoints without changes, and cluster heartbeats read nothing from the
// database and never rebuild the snapshot.
func TestCatalogUnchangedStateDoesNoWork(t *testing.T) {
	workspace, target := uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix)
	h := startCatalog(t, map[string]caller{
		uid.New(uid.DeploymentPrefix): callerOf(workspace, uid.New(uid.AppPrefix), target),
		uid.New(uid.DeploymentPrefix): callerOf(workspace, uid.New(uid.AppPrefix), target),
	})
	before := h.store.work()
	buildsBefore := builds(t)
	first := h.snapshot(t)

	cluster := []string{uid.New(uid.ClusterPrefix), uid.New(uid.RegionPrefix), "cell-1"}
	for i := range 100 {
		require.NoError(t, h.stream.deliver(t, heartbeat))
		require.NoError(t, h.stream.deliver(t, rowEvent("clusters", cluster, cluster)))
		require.NoError(t, h.stream.deliver(t, h.checkpoint("idle", time.Second)))
		h.clock.Tick(time.Second)
		snapshot := h.snapshot(t)
		require.Equal(t, first.Version, snapshot.Version, "iteration %d", i)
	}
	require.Equal(t, before, h.store.work())
	require.Equal(t, buildsBefore, builds(t))
}

// TestCatalogRefreshesOnlyAffectedCallers guarantees that one change reads
// only the callers whose entries it can change, and rebuilds the snapshot only
// when an entry changed.
func TestCatalogRefreshesOnlyAffectedCallers(t *testing.T) {
	workspace, otherWorkspace := uid.New(uid.WorkspacePrefix), uid.New(uid.WorkspacePrefix)
	api, dbApp, web, nobody := uid.New(uid.AppPrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	toDB, toWeb, webCaller := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	dbDeployment, lonely := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	h := startCatalog(t, map[string]caller{
		toDB:      callerOf(workspace, api, dbApp),
		toWeb:     callerOf(workspace, api, web),
		webCaller: callerOf(otherWorkspace, web, dbApp),
	})
	h.store.apps[dbDeployment] = dbApp

	for _, test := range []struct {
		name  string
		event cdc.Event
		want  []string
	}{
		{"target deployment change", rowEvent("deployments", []string{dbDeployment, dbApp}, []string{dbDeployment, dbApp}), []string{toDB, dbDeployment, webCaller}},
		{"target topology change", rowEvent("deployment_topology", []string{dbDeployment}, nil), []string{toDB, dbDeployment, webCaller}},
		{"target app change", rowEvent("apps", []string{web}, []string{web}), []string{toWeb, webCaller}},
		{"caller environment deletion", rowEvent("environments", []string{uid.New(uid.EnvironmentPrefix), api}, nil), []string{toDB, toWeb}},
		{"workspace deletion", rowEvent("workspaces", []string{otherWorkspace, "ns"}, nil), []string{webCaller}},
		{"unrelated deployment", rowEvent("deployments", []string{lonely, nobody}, []string{lonely, nobody}), []string{lonely}},
	} {
		t.Run(test.name, func(t *testing.T) {
			buildsBefore := builds(t)
			require.NoError(t, h.stream.deliver(t, test.event))
			require.NoError(t, h.stream.deliver(t, h.checkpoint(test.name, time.Second)))
			require.ElementsMatch(t, test.want, h.store.work().lastCallers)
			require.Equal(t, buildsBefore, builds(t), "entries read again unchanged")
		})
	}

	buildsBefore := builds(t)
	h.store.set(toWeb, callerOf(workspace, api, web, dbApp))
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{toWeb, api}, []string{toWeb, api})))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("changed", time.Second)))
	require.Equal(t, buildsBefore+1, builds(t))
}

// TestCatalogInvalidatesBothRowImages guarantees that a row leaving the CDC
// filter, a changed key column, and a change outside the selected columns
// each invalidate every key they can affect.
func TestCatalogInvalidatesBothRowImages(t *testing.T) {
	workspace, api, dbApp := uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	oldApp, newApp := uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	stopping, stoppingSecond, failing := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	toOld, toNew, moved := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	h := startCatalog(t, map[string]caller{
		stopping:       callerOf(workspace, api, dbApp),
		stoppingSecond: callerOf(workspace, api, dbApp),
		failing:        callerOf(workspace, api, dbApp),
		toOld:          callerOf(workspace, api, oldApp),
		toNew:          callerOf(workspace, api, newApp),
	})

	h.store.unset(stopping)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{stopping, api}, nil)))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("stopped", time.Second)))
	require.NotContains(t, callersIn(h.snapshot(t)), stopping, "a filtered running-to-stopped row with only a before image")

	h.store.unset(stoppingSecond)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{stoppingSecond, api}, []string{stoppingSecond, api})))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("stopped-2", time.Second)))
	require.NotContains(t, callersIn(h.snapshot(t)), stoppingSecond, "a running-to-stopped row with both images")

	h.store.unset(failing)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{failing, api}, []string{failing, api})))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("failed", time.Second)))
	require.ElementsMatch(t, []string{toNew, toOld}, callersIn(h.snapshot(t)), "a status change outside the selected columns")

	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{moved, oldApp}, []string{moved, newApp})))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("moved", time.Second)))
	require.ElementsMatch(t, []string{moved, toNew, toOld}, h.store.work().lastCallers)
}

// TestCatalogTopologyAndRebuilds guarantees a new copy when a change can make
// deployments active that no key identifies, a topology read when the
// topology's inputs change, and nothing for rewrites of identical inputs.
func TestCatalogTopologyAndRebuilds(t *testing.T) {
	workspace := uid.New(uid.WorkspacePrefix)
	region, otherRegion := uid.New(uid.RegionPrefix), uid.New(uid.RegionPrefix)
	cluster := uid.New(uid.ClusterPrefix)
	for _, test := range []struct {
		name     string
		event    cdc.Event
		topology bool
		rebuild  bool
	}{
		{"namespace assigned", rowEvent("workspaces", []string{workspace, ""}, []string{workspace, "ns"}), false, true},
		{"workspace added", rowEvent("workspaces", nil, []string{workspace, "ns"}), false, false},
		{"region platform changed", rowEvent("regions", []string{region, "aws", "us"}, []string{region, "gcp", "us"}), true, true},
		{"region deleted", rowEvent("regions", []string{region, "aws", "us"}, nil), true, true},
		{"region id changed", rowEvent("regions", []string{region, "aws", "us"}, []string{otherRegion, "aws", "us"}), true, true},
		{"region renamed", rowEvent("regions", []string{region, "aws", "us"}, []string{region, "aws", "eu"}), true, false},
		{"region added", rowEvent("regions", nil, []string{region, "aws", "us"}), true, false},
		{"region rewritten", rowEvent("regions", []string{region, "aws", "us"}, []string{region, "aws", "us"}), false, false},
		{"cluster added", rowEvent("clusters", nil, []string{cluster, region, "cell"}), true, false},
		{"cluster cell assigned", rowEvent("clusters", []string{cluster, region, ""}, []string{cluster, region, "cell"}), true, false},
		{"cluster heartbeat", rowEvent("clusters", []string{cluster, region, "cell"}, []string{cluster, region, "cell"}), false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := startCatalog(t, map[string]caller{uid.New(uid.DeploymentPrefix): callerOf(workspace, uid.New(uid.AppPrefix), uid.New(uid.AppPrefix))})
			topologies := h.store.work().topologyReads
			require.NoError(t, h.stream.deliver(t, test.event))
			err := h.stream.deliver(t, h.checkpoint("next", time.Second))
			if test.rebuild {
				require.ErrorIs(t, err, errRebuild)
				require.Nil(t, h.restarted(t), "a rebuild copies again")
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.topology, h.store.work().topologyReads > topologies)
		})
	}
}

// TestCatalogRefreshFailureKeepsCheckpoint guarantees that a change whose
// refresh failed is replayed, never skipped by an advanced checkpoint, and
// that the failure issues no snapshot ID.
func TestCatalogRefreshFailureKeepsCheckpoint(t *testing.T) {
	callerID, appID := uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix)
	h := startCatalog(t, map[string]caller{callerID: callerOf(uid.New(uid.WorkspacePrefix), appID, uid.New(uid.AppPrefix))})
	before := h.snapshot(t)
	h.clock.Tick(idInterval)
	h.store.failWith(errors.New("database unavailable"))
	h.store.unset(callerID)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{callerID, appID}, nil)))
	require.Error(t, h.stream.deliver(t, h.checkpoint("t1", time.Second)))
	require.Equal(t, before, h.snapshot(t), "a failed refresh changes neither content nor ID")

	require.Equal(t, []byte("t0"), h.restarted(t))
	h.store.failWith(nil)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{callerID, appID}, nil)))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("t1", time.Second)))
	after := h.snapshot(t)
	require.Empty(t, callersIn(after))
	require.NotEqual(t, before.ID, after.ID)

	h.stream.end(t, errors.New("stream lost"))
	require.Equal(t, []byte("t1"), h.restarted(t))
}

// TestCatalogCertifiesOnlyRecentCheckpoints guarantees that a snapshot counts
// toward a deletion only while its checkpoint is recent and has a commit time,
// so stalled, lagging, and sharded streams never authorize deletions.
func TestCatalogCertifiesOnlyRecentCheckpoints(t *testing.T) {
	firstCaller, secondCaller := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	workspace, appID, target := uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	h := startCatalog(t, map[string]caller{firstCaller: callerOf(workspace, appID, target), secondCaller: callerOf(workspace, appID, target)})
	first := h.snapshot(t)
	require.True(t, first.Certified)

	h.clock.Tick(certifiedAge)
	require.NoError(t, h.stream.deliver(t, heartbeat))
	stalled := h.snapshot(t)
	require.False(t, stalled.Certified, "heartbeats do not show that the stream caught up")
	require.Equal(t, first.ID, stalled.ID)

	h.store.unset(firstCaller)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{firstCaller, appID}, nil)))
	require.NoError(t, h.stream.deliver(t, h.checkpoint("lagging", 30*time.Second)))
	lagging := h.snapshot(t)
	require.False(t, lagging.Certified, "a lagging checkpoint publishes content without certifying it")
	require.Equal(t, []string{secondCaller}, callersIn(lagging))

	require.NoError(t, h.stream.deliver(t, h.checkpoint("future", -time.Second)))
	require.False(t, h.snapshot(t).Certified, "a source clock ahead of Ctrl cannot prove a recent past commit")
	h.clock.Tick(time.Second)
	require.True(t, h.snapshot(t).Certified, "certification starts at the commit time")
	h.clock.Tick(certifiedAge)
	require.True(t, h.snapshot(t).Certified, "certification includes its expiry boundary")
	h.clock.Tick(time.Nanosecond)
	require.False(t, h.snapshot(t).Certified, "certification ends after its expiry boundary")

	require.NoError(t, h.stream.deliver(t, h.checkpoint("caught-up", 0)))
	caughtUp := h.snapshot(t)
	require.True(t, caughtUp.Certified)
	require.NotEqual(t, first.ID, caughtUp.ID)
	require.Equal(t, lagging.Version, caughtUp.Version)

	require.NoError(t, h.stream.deliver(t, h.checkpoint("same-window", 0)))
	require.Equal(t, caughtUp.ID, h.snapshot(t).ID, "checkpoints in one window share an ID")

	require.NoError(t, h.stream.deliver(t, checkpointWithoutCommitTime("sharded")))
	sharded := h.snapshot(t)
	require.False(t, sharded.Certified, "a checkpoint without a commit time, as from a sharded keyspace, is never certified")
	require.Equal(t, caughtUp.ID, sharded.ID)
}

// TestCatalogDelayedCheckpointIsNotCertified guarantees that certification
// uses the time Ctrl reads the snapshot, so a checkpoint delivered late or
// applied after a slow refresh cannot certify the old position it reports.
func TestCatalogDelayedCheckpointIsNotCertified(t *testing.T) {
	firstCaller, secondCaller := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	workspace, appID, target := uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	h := startCatalog(t, map[string]caller{firstCaller: callerOf(workspace, appID, target), secondCaller: callerOf(workspace, appID, target)})

	queued := h.checkpoint("queued", 0)
	h.clock.Tick(certifiedAge + time.Second)
	require.NoError(t, h.stream.deliver(t, queued))
	before := h.snapshot(t)
	require.False(t, before.Certified, "a checkpoint delivered late")

	block := make(chan struct{})
	release := sync.OnceFunc(func() { close(block) })
	t.Cleanup(release)
	h.store.blockReads(block)
	h.store.unset(firstCaller)
	require.NoError(t, h.stream.deliver(t, rowEvent("deployments", []string{firstCaller, appID}, nil)))
	applied := h.stream.send(h.checkpoint("slow", 0))

	type observation struct {
		id, version string
		certified   bool
	}
	ctx, cancel := context.WithCancel(t.Context())
	var mu sync.Mutex
	seen := map[observation]int{}
	var readers sync.WaitGroup
	t.Cleanup(func() { cancel(); readers.Wait() })
	for range 4 {
		readers.Go(func() {
			for ctx.Err() == nil {
				snapshot, err := h.catalog.served.get()
				if err != nil {
					continue
				}
				mu.Lock()
				seen[observation{snapshot.ID, snapshot.Version, snapshot.Certified}]++
				mu.Unlock()
			}
		})
	}
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(seen) > 0
	}, time.Second, time.Millisecond)
	h.clock.Tick(certifiedAge + time.Second)
	mu.Lock()
	blocked := slices.Collect(maps.Keys(seen))
	mu.Unlock()
	require.Equal(t, []observation{{before.ID, before.Version, false}}, blocked, "a blocked refresh changes nothing")

	release()
	require.NoError(t, <-applied)
	after := h.snapshot(t)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return seen[observation{after.ID, after.Version, after.Certified}] > 0
	}, time.Second, time.Millisecond)
	cancel()
	readers.Wait()
	require.False(t, after.Certified, "a checkpoint applied after a slow refresh")
	require.Equal(t, []string{secondCaller}, callersIn(after))
	for o := range seen {
		require.Contains(t, []observation{{before.ID, before.Version, false}, {after.ID, after.Version, false}}, o)
	}
}

// TestCatalogIDsAgreeAcrossInstances guarantees that several Ctrl instances, a
// restarted Ctrl, and a lagging Ctrl cannot give Krane two certified IDs for
// one position, or certify a copy before it applied a recent checkpoint.
func TestCatalogIDsAgreeAcrossInstances(t *testing.T) {
	active := map[string]caller{uid.New(uid.DeploymentPrefix): callerOf(uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix))}
	running := startCatalog(t, active)
	restarted := runCatalog(t, active)
	restarted.copy(t, active)
	copied := restarted.snapshot(t)
	require.False(t, copied.Certified, "a copy is not certified before it applies a checkpoint")
	require.NotEqual(t, running.snapshot(t).ID, copied.ID)

	position := cdc.Event{ResumeToken: []byte("t1"), CommitTime: running.clock.Now()}
	require.NoError(t, running.stream.deliver(t, position))
	require.NoError(t, restarted.stream.deliver(t, position))
	first, second := running.snapshot(t), restarted.snapshot(t)
	require.True(t, first.Certified)
	require.True(t, second.Certified)
	require.Equal(t, first.ID, second.ID, "instances at one position report one ID")
	require.Equal(t, first.Version, second.Version)

	lagging := runCatalog(t, active)
	lagging.copy(t, active)
	lagging.clock.Tick(certifiedAge + time.Second)
	require.NoError(t, lagging.stream.deliver(t, position))
	late := lagging.snapshot(t)
	require.False(t, late.Certified, "an instance that applies the position late does not certify it")
	require.Equal(t, first.ID, late.ID)
}

// TestCatalogServesWhileStreamIsLive guarantees that a silent stream is not
// served, and that a rebuild keeps serving the previous snapshot under its ID
// until the new copy completes.
func TestCatalogServesWhileStreamIsLive(t *testing.T) {
	firstCaller, secondCaller := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	workspace, appID, target := uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	active := map[string]caller{firstCaller: callerOf(workspace, appID, target)}
	h := startCatalog(t, active)
	before := h.snapshot(t)

	h.clock.Tick(liveWindow + time.Second)
	_, err := h.catalog.served.get()
	require.ErrorIs(t, err, ErrUnavailable, "a silent stream is not served")
	require.NoError(t, h.stream.deliver(t, heartbeat))
	require.Equal(t, before.Version, h.snapshot(t).Version)

	h.stream.end(t, cdc.ErrExpired)
	require.Nil(t, h.restarted(t), "an expired position copies again")
	require.NoError(t, h.stream.deliver(t, heartbeat))
	during := h.snapshot(t)
	require.Equal(t, before.ID, during.ID, "a rebuild serves the previous snapshot")
	require.Equal(t, before.Version, during.Version)

	active[secondCaller] = callerOf(workspace, appID, target)
	h.store.set(secondCaller, active[secondCaller])
	h.copy(t, active)
	rebuilt := h.snapshot(t)
	require.Equal(t, before.ID, rebuilt.ID, "a copy completion issues no ID")
	require.False(t, rebuilt.Certified)
	require.ElementsMatch(t, []string{firstCaller, secondCaller}, callersIn(rebuilt))
}

// TestCatalogSnapshotsAreSafeDuringRefresh guarantees that request goroutines
// read published snapshots while the stream goroutine refreshes.
func TestCatalogSnapshotsAreSafeDuringRefresh(t *testing.T) {
	workspace, appID, target := uid.New(uid.WorkspacePrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	h := startCatalog(t, map[string]caller{uid.New(uid.DeploymentPrefix): callerOf(workspace, appID, target)})
	ctx, cancel := context.WithCancel(t.Context())
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for ctx.Err() == nil {
				snapshot, err := h.catalog.served.wait(ctx)
				if err == nil && len(snapshot.Connections) > 0 {
					_ = snapshot.Connections[len(snapshot.Connections)-1].GetConnectionId()
				}
			}
		})
	}
	callerIDs := make([]string, 5)
	for i := range callerIDs {
		callerIDs[i] = uid.New(uid.DeploymentPrefix)
	}
	for i := range 50 {
		id := callerIDs[i%len(callerIDs)]
		h.store.set(id, callerOf(workspace, appID, target))
		require.NoError(t, h.stream.deliver(t, rowEvent("deployments", nil, []string{id, appID})))
		require.NoError(t, h.stream.deliver(t, h.checkpoint(id, time.Second)))
		h.clock.Tick(time.Second)
	}
	cancel()
	readers.Wait()
}
