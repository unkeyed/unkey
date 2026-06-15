package clickhouse_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	gib       = int64(1024 * 1024 * 1024)
	sampleGap = int64(15_000) // 15s in ms, heimdall's tick
)

// gibHours converts a GiB-millisecond integral into GiB-hours, matching the
// query's `/ 1000 / 3600 / pow(1024,3)` scaling but starting from a GiB value
// already divided out, so callers express the expectation as gib * ms.
func gibHours(byteMillis float64) float64 {
	return byteMillis / 1000 / 3600 / float64(gib)
}

// findUsage returns the row for a resource, or fails if absent.
func findUsage(t *testing.T, rows []clickhouse.InstanceMeterUsage, resourceID string) clickhouse.InstanceMeterUsage {
	t.Helper()
	for _, r := range rows {
		if r.ResourceID == resourceID {
			return r
		}
	}
	t.Fatalf("no usage row for resource %q (got %d rows)", resourceID, len(rows))
	return clickhouse.InstanceMeterUsage{}
}

func TestGetInstanceMeterUsage(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)

	client, err := clickhouse.New(clickhouse.Config{URL: cfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	require.NoError(t, conn.Ping(ctx))

	// Anchor every checkpoint to the start of yesterday (UTC): deterministic
	// partition/ts math within a run, but derived from now so the rows always
	// sit far inside the table's 95-day TTL. A hardcoded date would silently
	// age out and ClickHouse would drop the parts before the query runs.
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour).UnixMilli()
	windowStart := base - sampleGap
	windowEnd := base + int64(time.Hour/time.Millisecond)

	t.Run("counter deltas and time integration over a clean series", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		resource := uid.New("res")
		container := newContainer()

		// 5 samples, 15s apart. CPU and egress are monotonic counters;
		// memory and disk are flat gauges.
		var samples []schema.InstanceCheckpoint
		for i := 0; i < 5; i++ {
			samples = append(samples, container.sample(ws, resource, base+int64(i)*sampleGap, sampleValues{
				cpuUsec:     int64(i) * 1_000_000, // 1 CPU-second per step
				egressBytes: int64(i) * 1000,
				memoryBytes: gib,     // 1 GiB
				diskBytes:   2 * gib, // 2 GiB
			}))
		}
		insertCheckpoints(t, ctx, conn, samples)

		rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: ws,
			Start:       windowStart,
			End:         windowEnd,
		})
		require.NoError(t, err)

		u := findUsage(t, rows, resource)
		// 4 deltas of 1e6 usec = 4 CPU-seconds.
		require.InDelta(t, 4.0, u.CPUSeconds, 1e-9)
		// 4 deltas of 1000 bytes.
		require.Equal(t, int64(4000), u.EgressBytes)
		// 4 pairs * 15000ms * 1 GiB.
		require.InDelta(t, gibHours(float64(gib)*float64(4*sampleGap)), u.MemoryGiBHours, 1e-9)
		// Same intervals, 2 GiB allocated.
		require.InDelta(t, gibHours(float64(2*gib)*float64(4*sampleGap)), u.DiskGiBHours, 1e-9)
	})

	t.Run("drops samples more than two minutes apart", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		resource := uid.New("res")
		container := newContainer()

		// Three samples close together, then a 5-minute outage, then two more.
		// The counter keeps climbing across the outage (cgroup counters don't
		// pause), but the spanning pair must be dropped, so its delta is not
		// billed.
		offsets := []int64{0, sampleGap, 2 * sampleGap, 2*sampleGap + 5*60*1000, 2*sampleGap + 5*60*1000 + sampleGap}
		var samples []schema.InstanceCheckpoint
		for i, off := range offsets {
			samples = append(samples, container.sample(ws, resource, base+off, sampleValues{
				cpuUsec:     int64(i) * 1_000_000,
				egressBytes: int64(i) * 1000,
				memoryBytes: gib,
				diskBytes:   gib,
			}))
		}
		insertCheckpoints(t, ctx, conn, samples)

		rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: ws,
			Start:       windowStart,
			End:         windowEnd,
		})
		require.NoError(t, err)

		u := findUsage(t, rows, resource)
		// Kept pairs: (0,1),(1,2),(3,4) = 3 deltas of 1e6. The (2,3) pair
		// across the 5-min gap is dropped.
		require.InDelta(t, 3.0, u.CPUSeconds, 1e-9)
		require.Equal(t, int64(3000), u.EgressBytes)
		// Memory integrates only the 3 kept 15s intervals.
		require.InDelta(t, gibHours(float64(gib)*float64(3*sampleGap)), u.MemoryGiBHours, 1e-9)
	})

	t.Run("never diffs counters across a restart boundary", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		resource := uid.New("res")

		// Restart 0: cpu climbs 0 -> 3e6 over 3 samples.
		c0 := newContainerWithRestart(0)
		var samples []schema.InstanceCheckpoint
		for i := 0; i < 3; i++ {
			samples = append(samples, c0.sample(ws, resource, base+int64(i)*sampleGap, sampleValues{
				cpuUsec:     int64(i) * 1_000_000,
				egressBytes: int64(i) * 500,
				memoryBytes: gib,
				diskBytes:   gib,
			}))
		}
		// Restart 1: fresh cgroup, counter resets to 0 and climbs again. A
		// naive global max-min across the boundary would go negative or
		// double-count; per-container_uid integration must not.
		c1 := newContainerWithRestart(1)
		restartBase := base + 3*sampleGap
		for i := 0; i < 3; i++ {
			samples = append(samples, c1.sample(ws, resource, restartBase+int64(i)*sampleGap, sampleValues{
				cpuUsec:     int64(i) * 1_000_000,
				egressBytes: int64(i) * 500,
				memoryBytes: gib,
				diskBytes:   gib,
			}))
		}
		insertCheckpoints(t, ctx, conn, samples)

		rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: ws,
			Start:       windowStart,
			End:         windowEnd,
		})
		require.NoError(t, err)

		u := findUsage(t, rows, resource)
		// Each life: 2 deltas of 1e6 = 2 CPU-seconds. Two lives = 4.
		require.InDelta(t, 4.0, u.CPUSeconds, 1e-9)
		require.Equal(t, int64(2000), u.EgressBytes)
	})

	t.Run("empty workspace id aggregates across workspaces", func(t *testing.T) {
		// Two distinct workspaces, queried with an empty filter, both appear.
		wsA := uid.New(uid.WorkspacePrefix)
		wsB := uid.New(uid.WorkspacePrefix)
		resA := uid.New("res")
		resB := uid.New("res")

		var samples []schema.InstanceCheckpoint
		ca := newContainer()
		cb := newContainer()
		for i := 0; i < 2; i++ {
			samples = append(samples, ca.sample(wsA, resA, base+int64(i)*sampleGap, sampleValues{
				cpuUsec: int64(i) * 2_000_000, memoryBytes: gib, diskBytes: gib,
			}))
			samples = append(samples, cb.sample(wsB, resB, base+int64(i)*sampleGap, sampleValues{
				cpuUsec: int64(i) * 7_000_000, memoryBytes: gib, diskBytes: gib,
			}))
		}
		insertCheckpoints(t, ctx, conn, samples)

		rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
			Start: windowStart,
			End:   windowEnd,
		})
		require.NoError(t, err)

		require.InDelta(t, 2.0, findUsage(t, rows, resA).CPUSeconds, 1e-9)
		require.InDelta(t, 7.0, findUsage(t, rows, resB).CPUSeconds, 1e-9)
	})

	t.Run("returns no rows outside the window", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		resource := uid.New("res")
		c := newContainer()
		samples := []schema.InstanceCheckpoint{
			c.sample(ws, resource, base, sampleValues{cpuUsec: 0, memoryBytes: gib, diskBytes: gib}),
			c.sample(ws, resource, base+sampleGap, sampleValues{cpuUsec: 1_000_000, memoryBytes: gib, diskBytes: gib}),
		}
		insertCheckpoints(t, ctx, conn, samples)

		// Query a window a full day after the samples.
		rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: ws,
			Start:       base + int64(24*time.Hour/time.Millisecond),
			End:         base + int64(48*time.Hour/time.Millisecond),
		})
		require.NoError(t, err)
		require.Empty(t, rows)
	})
}

// sampleValues holds the per-checkpoint metric values a test wants to set; the
// rest of the row is boilerplate filled in by container.sample.
type sampleValues struct {
	cpuUsec     int64
	egressBytes int64
	memoryBytes int64
	diskBytes   int64
}

// container builds checkpoints for one container lifecycle, holding the stable
// identity (pod_uid, restart_count) so a test doesn't repeat it per sample.
type container struct {
	podUID       string
	instanceID   string
	restartCount uint32
}

func newContainer() *container { return newContainerWithRestart(0) }

func newContainerWithRestart(restart uint32) *container {
	return &container{
		podUID:       uid.New("pod"),
		instanceID:   uid.New("inst"),
		restartCount: restart,
	}
}

func (c *container) sample(ws, resource string, ts int64, v sampleValues) schema.InstanceCheckpoint {
	return schema.InstanceCheckpoint{
		NodeID:                   "node-1",
		WorkspaceID:              ws,
		ProjectID:                "proj_test",
		EnvironmentID:            "env_test",
		ResourceType:             "deployment",
		ResourceID:               resource,
		PodUID:                   c.podUID,
		InstanceID:               c.instanceID,
		ContainerUID:             c.podUID + "/" + strconv.Itoa(int(c.restartCount)),
		RestartCount:             c.restartCount,
		Ts:                       ts,
		EventKind:                "checkpoint",
		CPUUsageUsec:             v.cpuUsec,
		MemoryBytes:              v.memoryBytes,
		DiskAllocatedBytes:       v.diskBytes,
		NetworkEgressPublicBytes: v.egressBytes,
		Region:                   "local",
		Platform:                 "local",
		Attributes:               "{}",
	}
}

func insertCheckpoints(t *testing.T, ctx context.Context, conn ch.Conn, samples []schema.InstanceCheckpoint) {
	t.Helper()
	if len(samples) == 0 {
		return
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO default.instance_checkpoints_v1")
	require.NoError(t, err)
	for i := range samples {
		require.NoError(t, batch.AppendStruct(&samples[i]))
	}
	require.NoError(t, batch.Send())
}
