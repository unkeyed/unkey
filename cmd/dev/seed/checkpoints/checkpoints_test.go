package checkpoints

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// collector captures buffered checkpoints in memory so a test can replay the
// billing query against them.
type collector struct{ rows []schema.InstanceCheckpoint }

func (c *collector) Buffer(cp schema.InstanceCheckpoint) { c.rows = append(c.rows, cp) }

// billFromRows replays the authoritative Deploy billing aggregation
// (pkg/clickhouse GetInstanceMeterUsage / web deploy_billing.ts): per
// container_uid, integrate over consecutive checkpoint pairs whose gap is
// within the billing window.
func billFromRows(rows []schema.InstanceCheckpoint) billable {
	const maxGapMs = int64(2 * 60 * 1000)
	const gib = 1024 * 1024 * 1024

	byContainer := map[string][]schema.InstanceCheckpoint{}
	for _, r := range rows {
		byContainer[r.ContainerUID] = append(byContainer[r.ContainerUID], r)
	}

	var cpuUsec, egressBytes int64
	var memByteMs, diskByteMs float64
	for _, series := range byContainer {
		sort.Slice(series, func(i, j int) bool { return series[i].Ts < series[j].Ts })
		for i := 0; i+1 < len(series); i++ {
			cur, next := series[i], series[i+1]
			dt := next.Ts - cur.Ts
			if dt <= 0 || dt > maxGapMs {
				continue
			}
			if d := next.CPUUsageUsec - cur.CPUUsageUsec; d > 0 {
				cpuUsec += d
			}
			if d := next.NetworkEgressPublicBytes - cur.NetworkEgressPublicBytes; d > 0 {
				egressBytes += d
			}
			memByteMs += float64(minInt64(cur.MemoryBytes, next.MemoryBytes)) * float64(dt)
			diskByteMs += float64(minInt64(cur.DiskAllocatedBytes, next.DiskAllocatedBytes)) * float64(dt)
		}
	}

	return billable{
		cpuSeconds:     float64(cpuUsec) / 1e6,
		memGiBSeconds:  memByteMs / 1000 / gib,
		diskGiBSeconds: diskByteMs / 1000 / gib,
		egressGiB:      float64(egressBytes) / gib,
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func TestGeneratorMatchesBillingMath(t *testing.T) {
	cases := []struct {
		name string
		g    generator
	}{
		{
			name: "24/7 single replica",
			g: generator{ //nolint:exhaustruct
				target:        target{workspaceID: "ws", projectID: "proj", environmentID: "env", resourceID: "d_1"}, //nolint:exhaustruct
				vcpu:          0.5,
				memoryBytes:   512 * 1024 * 1024,
				diskBytes:     1024 * 1024 * 1024,
				egressPerDay:  2 * 1024 * 1024 * 1024,
				replicas:      1,
				days:          7,
				hoursPerDay:   24,
				tick:          60 * time.Second,
				cpuAllocMilli: 1000,
				memAllocBytes: 1024 * 1024 * 1024,
			},
		},
		{
			name: "partial uptime, multi replica",
			g: generator{ //nolint:exhaustruct
				target:        target{workspaceID: "ws", projectID: "proj", environmentID: "env", resourceID: "d_2"}, //nolint:exhaustruct
				vcpu:          2,
				memoryBytes:   1024 * 1024 * 1024,
				diskBytes:     0,
				egressPerDay:  500 * 1024 * 1024,
				replicas:      3,
				days:          5,
				hoursPerDay:   8,
				tick:          30 * time.Second,
				cpuAllocMilli: 2000,
				memAllocBytes: 2 * 1024 * 1024 * 1024,
			},
		},
		{
			// Fractional vCPU and a non-power-of-two egress: exercises the
			// index-based counters, which must still bill exactly what
			// expected() predicts (no compounding per-tick truncation).
			name: "fractional vcpu",
			g: generator{ //nolint:exhaustruct
				target:        target{workspaceID: "ws", projectID: "proj", environmentID: "env", resourceID: "d_3"}, //nolint:exhaustruct
				vcpu:          0.333,
				memoryBytes:   256 * 1024 * 1024,
				diskBytes:     0,
				egressPerDay:  1_500_000_000,
				replicas:      2,
				days:          10,
				hoursPerDay:   24,
				tick:          45 * time.Second,
				cpuAllocMilli: 500,
				memAllocBytes: 512 * 1024 * 1024,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.g
			g.end = time.Now()
			c := &collector{}
			rows := g.generate(c)
			require.Equal(t, rows, len(c.rows))
			require.NotEmpty(t, c.rows)

			got := billFromRows(c.rows)
			want := g.expected()

			require.InEpsilon(t, want.cpuSeconds, got.cpuSeconds, 1e-9, "cpuSeconds")
			require.InEpsilon(t, want.memGiBSeconds, got.memGiBSeconds, 1e-9, "memGiBSeconds")
			require.InEpsilon(t, want.egressGiB, got.egressGiB, 1e-9, "egressGiB")
			if want.diskGiBSeconds == 0 {
				require.Zero(t, got.diskGiBSeconds)
			} else {
				require.InEpsilon(t, want.diskGiBSeconds, got.diskGiBSeconds, 1e-9, "diskGiBSeconds")
			}
		})
	}
}

func TestParseBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"1024", 1024},
		{"512Mi", 512 * 1024 * 1024},
		{"1Gi", 1024 * 1024 * 1024},
		{"2Ti", 2 * 1024 * 1024 * 1024 * 1024},
		{"1G", 1_000_000_000},
		{"500M", 500_000_000},
		{"1B", 1},
	}
	for _, tc := range cases {
		got, err := parseBytes(tc.in)
		require.NoError(t, err, tc.in)
		require.Equal(t, tc.want, got, tc.in)
	}

	_, err := parseBytes("abc")
	require.Error(t, err)
}
