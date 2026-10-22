package privatenetwork

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
	"google.golang.org/protobuf/proto"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

const (
	idInterval   = 5 * time.Second
	certifiedAge = 10 * time.Second
	liveWindow   = 15 * time.Second
	minBackoff   = time.Second
	maxBackoff   = 30 * time.Second

	refreshBootstrap = "bootstrap"
	refreshUpdate    = "update"
)

// ErrUnavailable means the catalog has not been built yet or its stream has
// been silent for too long.
var ErrUnavailable = errors.New("private network catalog is not available")

var errRebuild = errors.New("private network catalog needs a new snapshot")

// Snapshot is the complete private network catalog of one platform.
type Snapshot struct {
	Connections []*ctrlv1.PrivateNetworkConnection
	Topology    *ctrlv1.PrivateNetworkTopology
	// Version is a digest of Connections and Topology.
	Version string
	// ID names the five-second window in which the last applied CDC
	// checkpoint committed, so every Ctrl instance that applied the same
	// checkpoint reports the same ID. Snapshots with different IDs reflect
	// different database positions. Until a checkpoint with a known commit
	// time is applied, the ID is random and the snapshot is not certified.
	ID string
	// Certified reports that the last applied checkpoint committed at most ten
	// seconds ago by Ctrl's clock, which assumes the database and Ctrl clocks
	// agree to within a few seconds. Krane counts an omission toward deleting
	// a grant only in certified snapshots with two IDs. A keyspace with more
	// than one shard never reports a commit time, so it is never certified and
	// Krane never deletes grants for it.
	Certified bool
}

type watchFunc func(ctx context.Context, token []byte, send func(cdc.Event) error) error

type index map[string]set

func (i index) add(key, deploymentID string) {
	if key == "" {
		return
	}
	if i[key] == nil {
		i[key] = set{}
	}
	i[key].add(deploymentID)
}

func (i index) remove(key, deploymentID string) {
	delete(i[key], deploymentID)
	if len(i[key]) == 0 {
		delete(i, key)
	}
}

type catalog struct {
	platform string
	store    store
	watch    watchFunc
	clock    clock.Clock
	served   *publication

	callers     map[string]caller
	targeting   index
	byApp       index
	byWorkspace index
	topology    *ctrlv1.PrivateNetworkTopology
	pending     changes
	copied      bool
	token       []byte
	content     Snapshot
	id          string
}

func newCatalog(platform string, s store, watch watchFunc, clk clock.Clock) *catalog {
	c := &catalog{
		platform:    platform,
		store:       s,
		watch:       watch,
		clock:       clk,
		served:      newPublication(platform, clk),
		callers:     nil,
		targeting:   nil,
		byApp:       nil,
		byWorkspace: nil,
		topology:    nil,
		pending:     newChanges(),
		copied:      false,
		token:       nil,
		content:     Snapshot{Connections: nil, Topology: nil, Version: "", ID: "", Certified: false},
		id:          "",
	}
	c.reset()
	return c
}

func (c *catalog) run(ctx context.Context) {
	backoff := minBackoff
	for {
		progressed := false
		err := c.watch(ctx, c.token, func(event cdc.Event) error {
			advanced, err := c.apply(ctx, event)
			progressed = progressed || advanced
			return err
		})
		if ctx.Err() != nil {
			return
		}

		restart := "resume"
		if c.token == nil || errors.Is(err, errRebuild) || errors.Is(err, cdc.ErrInvalidToken) || errors.Is(err, cdc.ErrExpired) {
			restart = "rebuild"
			c.reset()
		}
		metrics.PrivateNetworkCatalogRestartsTotal.WithLabelValues(restart).Inc()
		logger.Warn("private network catalog stream ended", "platform", c.platform, "restart", restart, "error", err)
		if progressed {
			backoff = minBackoff
		}
		if !c.sleep(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func (c *catalog) sleep(ctx context.Context, d time.Duration) bool {
	ticker := c.clock.NewTicker(d)
	defer ticker.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-ticker.C():
		return true
	}
}

func (c *catalog) apply(ctx context.Context, event cdc.Event) (advanced bool, err error) {
	c.served.markLive()

	switch {
	case event.Change != nil:
		if event.Change.Type != binlog.VEventType_ROW {
			return false, nil
		}
		rows, err := decodeRows(event.Change)
		if err != nil {
			return false, err
		}
		for _, row := range rows {
			c.pending.add(row)
		}
		return false, nil
	case event.Heartbeat:
		return false, nil
	case event.CopyCompleted:
		if c.copied {
			return false, errors.New("private network catalog received a second copy completion")
		}
		if _, err := c.refresh(ctx, refreshBootstrap); err != nil {
			return false, err
		}
		if err := c.build(); err != nil {
			return false, err
		}
		if c.id == "" {
			c.id = uid.New("snapshot")
		}
		c.publish(time.Time{})
		c.copied = true
		return true, nil
	default:
		if !c.copied {
			return false, nil
		}
		changed, err := c.refresh(ctx, refreshUpdate)
		if err != nil {
			return false, err
		}
		if changed {
			if err := c.build(); err != nil {
				return false, err
			}
		}
		certifiedUntil := time.Time{}
		if !event.CommitTime.IsZero() {
			c.id = "cdc_" + strconv.FormatInt(event.CommitTime.Unix()/int64(idInterval/time.Second), 10)
			certifiedUntil = event.CommitTime.Add(certifiedAge)
		}
		c.publish(certifiedUntil)
		c.token = event.ResumeToken
		return true, nil
	}
}

func (c *catalog) refresh(ctx context.Context, kind string) (bool, error) {
	if c.pending.rebuild {
		return false, errRebuild
	}
	if kind == refreshUpdate && c.pending.empty() {
		return false, nil
	}
	started := c.clock.Now()
	affected := set{}
	maps.Copy(affected, c.pending.callers)
	if kind == refreshUpdate {
		if len(c.pending.topologyCallers) > 0 {
			apps, err := c.store.deploymentApps(ctx, slices.Sorted(maps.Keys(c.pending.topologyCallers)))
			if err != nil {
				return false, err
			}
			for _, app := range apps {
				c.pending.targetApps.add(app)
			}
		}
		for app := range c.pending.targetApps {
			maps.Copy(affected, c.targeting[app])
		}
		for app := range c.pending.callerApps {
			maps.Copy(affected, c.byApp[app])
		}
		for workspace := range c.pending.workspaces {
			maps.Copy(affected, c.byWorkspace[workspace])
		}
	}

	active := map[string]caller{}
	if len(affected) > 0 {
		var err error
		active, err = c.store.activeCallers(ctx, c.platform, slices.Sorted(maps.Keys(affected)))
		if err != nil {
			return false, err
		}
	}
	topology := c.topology
	if kind == refreshBootstrap || c.pending.topology {
		var err error
		topology, err = c.store.topology(ctx, c.platform)
		if err != nil {
			return false, err
		}
	}

	changed := !proto.Equal(topology, c.topology)
	for deploymentID := range affected {
		previous, had := c.callers[deploymentID]
		entry, ok := active[deploymentID]
		if had == ok && (!ok || sameCaller(previous, entry)) {
			continue
		}
		changed = true
		c.removeCaller(deploymentID)
		if ok {
			c.addCaller(deploymentID, entry)
		}
	}
	c.topology = topology
	c.pending = newChanges()
	metrics.PrivateNetworkCatalogRefreshesTotal.WithLabelValues(kind).Inc()
	metrics.PrivateNetworkCatalogRefreshedCallersTotal.WithLabelValues(kind).Add(float64(len(affected)))
	metrics.PrivateNetworkCatalogRefreshDurationSeconds.WithLabelValues(kind).Observe(c.clock.Now().Sub(started).Seconds())
	return changed, nil
}

func sameCaller(a, b caller) bool {
	equal := func(x, y *ctrlv1.PrivateNetworkConnection) bool { return proto.Equal(x, y) }
	return a.workspaceID == b.workspaceID && a.appID == b.appID &&
		equal(a.replica, b.replica) && slices.EqualFunc(a.connections, b.connections, equal)
}

func (c *catalog) addCaller(deploymentID string, entry caller) {
	c.callers[deploymentID] = entry
	c.byApp.add(entry.appID, deploymentID)
	c.byWorkspace.add(entry.workspaceID, deploymentID)
	for _, connection := range entry.connections {
		c.targeting.add(connection.GetTargetAppId(), deploymentID)
	}
}

func (c *catalog) removeCaller(deploymentID string) {
	entry, ok := c.callers[deploymentID]
	if !ok {
		return
	}
	delete(c.callers, deploymentID)
	c.byApp.remove(entry.appID, deploymentID)
	c.byWorkspace.remove(entry.workspaceID, deploymentID)
	for _, connection := range entry.connections {
		c.targeting.remove(connection.GetTargetAppId(), deploymentID)
	}
}

func (c *catalog) publish(certifiedUntil time.Time) {
	snapshot := c.content
	snapshot.ID = c.id
	c.served.set(snapshot, certifiedUntil)
}

func (c *catalog) build() error {
	metrics.PrivateNetworkCatalogBuildsTotal.Inc()
	var connections, replicas []*ctrlv1.PrivateNetworkConnection
	for _, entry := range c.callers {
		connections = append(connections, entry.connections...)
		if entry.replica != nil {
			replicas = append(replicas, entry.replica)
		}
	}
	slices.SortFunc(connections, compareConnections)
	slices.SortFunc(replicas, compareConnections)
	connections = append(connections, replicas...)

	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(&ctrlv1.PrivateNetworkStateChunk{
		Connections: connections,
		Topology:    c.topology,
	})
	if err != nil {
		return err
	}
	digest := sha256.Sum256(encoded)
	c.content = Snapshot{Connections: connections, Topology: c.topology, Version: hex.EncodeToString(digest[:]), ID: "", Certified: false}
	return nil
}

func compareConnections(a, b *ctrlv1.PrivateNetworkConnection) int {
	return cmp.Or(
		strings.Compare(a.GetCallerDeploymentId(), b.GetCallerDeploymentId()),
		strings.Compare(a.GetConnectionId(), b.GetConnectionId()),
	)
}

func (c *catalog) reset() {
	c.callers = map[string]caller{}
	c.targeting = index{}
	c.byApp = index{}
	c.byWorkspace = index{}
	c.topology = nil
	c.pending = newChanges()
	c.copied = false
	c.token = nil
}
