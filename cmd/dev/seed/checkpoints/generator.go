package checkpoints

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
)

// generator emits the checkpoint series and tracks the resulting billable
// totals so callers can assert the billing pipeline computes the same numbers.
type generator struct {
	target       target
	vcpu         float64
	memoryBytes  int64
	diskBytes    int64
	egressPerDay int64
	replicas     int
	days         int
	hoursPerDay  float64
	tick         time.Duration
	end          time.Time
	region       string
	platform     string

	cpuAllocMilli int32
	memAllocBytes int64
}

// bufferer is the subset of clickhouse's batch processor the generator needs;
// narrowed to an interface so tests can capture rows in memory.
type bufferer interface {
	Buffer(schema.InstanceCheckpoint)
}

// run is a contiguous active window [startMs, endMs] during which checkpoints
// are emitted every tick. Runs are separated by gaps longer than the billing
// gap, so usage during downtime is correctly unbilled.
type run struct{ startMs, endMs int64 }

// runs returns the active windows. A 24/7 instance is one continuous run over
// the whole window (no per-day boundaries, so no duplicate-timestamp rows).
// A partial-uptime instance is one gapped run per day.
func (g generator) runs() []run {
	dayMs := int64(24 * time.Hour / time.Millisecond)
	blockMs := int64(g.hoursPerDay * float64(time.Hour/time.Millisecond))
	endMs := g.end.UnixMilli()
	startMs := endMs - int64(g.days)*dayMs

	if g.hoursPerDay >= 24 {
		return []run{{startMs: startMs, endMs: endMs}}
	}
	out := make([]run, 0, g.days)
	for d := 0; d < g.days; d++ {
		dayStart := startMs + int64(d)*dayMs
		out = append(out, run{startMs: dayStart, endMs: dayStart + blockMs})
	}
	return out
}

// intervalsPerDay is the number of billable tick intervals in one active day.
func (g generator) intervalsPerDay() int64 {
	blockMs := int64(g.hoursPerDay * float64(time.Hour/time.Millisecond))
	return blockMs / g.tick.Milliseconds()
}

// generate writes the checkpoint series to buf and returns the row count.
func (g generator) generate(buf bufferer) int {
	tickMs := g.tick.Milliseconds()
	intervalsPerDay := g.intervalsPerDay()
	// cpuUsecPerDay is the CPU microseconds consumed in one active day. Like
	// egress it is truncated to int64 once (sub-microsecond) and then spread by
	// tick index, so a fractional --vcpu can't compound a per-tick rounding.
	cpuUsecPerDay := int64(g.vcpu * float64(intervalsPerDay) * float64(tickMs) * 1000)
	runs := g.runs()

	count := 0
	for r := 0; r < g.replicas; r++ {
		podUID := uid.New("pod")
		containerUID := podUID + "/0"
		instanceID := fmt.Sprintf("%s-%d", g.target.resourceID, r)

		// Counters are monotonic across the container's whole lifetime. Cross-run
		// pairs are gap-dropped by the billing query, so carrying the value over
		// a gap is harmless (the delta is zero anyway: no ticks run in the gap).
		//
		// CPU and egress are derived from the tick index rather than accumulated
		// per tick, so integer truncation never compounds: at tickInDay ==
		// intervalsPerDay the day closes on exactly cpuUsecPerDay / egressPerDay.
		// The *Base values carry completed days so the counters stay monotonic.
		var cpuBase, egressBase int64
		for _, rn := range runs {
			tickInDay := int64(0)
			for ts := rn.startMs; ts <= rn.endMs; ts += tickMs {
				cpuUsec, egressBytes := cpuBase, egressBase
				if intervalsPerDay > 0 {
					cpuUsec += cpuUsecPerDay * tickInDay / intervalsPerDay
					egressBytes += g.egressPerDay * tickInDay / intervalsPerDay
				}
				buf.Buffer(schema.InstanceCheckpoint{
					NodeID:                     "seed-node",
					WorkspaceID:                g.target.workspaceID,
					ProjectID:                  g.target.projectID,
					EnvironmentID:              g.target.environmentID,
					ResourceType:               "deployment",
					ResourceID:                 g.target.resourceID,
					PodUID:                     podUID,
					InstanceID:                 instanceID,
					ContainerUID:               containerUID,
					RestartCount:               0,
					Ts:                         ts,
					EventKind:                  "checkpoint",
					CPUUsageUsec:               cpuUsec,
					MemoryBytes:                g.memoryBytes,
					CPUAllocatedMillicores:     g.cpuAllocMilli,
					MemoryAllocatedBytes:       g.memAllocBytes,
					DiskAllocatedBytes:         g.diskBytes,
					DiskUsedBytes:              g.diskBytes,
					NetworkEgressPublicBytes:   egressBytes,
					NetworkEgressPrivateBytes:  0,
					NetworkIngressPublicBytes:  0,
					NetworkIngressPrivateBytes: 0,
					Region:                     g.region,
					Platform:                   g.platform,
					Attributes:                 "{}",
				})
				count++
				tickInDay++
			}
			// Carry the run's full usage (tickInDay-1 == intervalsPerDay at the
			// last emit of a complete day) so the next run continues monotonically.
			if intervalsPerDay > 0 {
				cpuBase += cpuUsecPerDay * (tickInDay - 1) / intervalsPerDay
				egressBase += g.egressPerDay * (tickInDay - 1) / intervalsPerDay
			}
		}
	}
	return count
}

// billable is the per-meter usage the billing pipeline should compute from the
// generated series.
type billable struct {
	cpuSeconds     float64
	memGiBSeconds  float64
	diskGiBSeconds float64
	egressGiB      float64
}

// expected predicts the billable totals from the same runs/tick math the
// generator emits, so callers can assert the pipeline agrees.
func (g generator) expected() billable {
	tickMs := g.tick.Milliseconds()
	tickSeconds := float64(tickMs) / 1000.0

	// Billing telescopes over consecutive in-run pairs; each run of N points
	// contributes N-1 intervals of one tick.
	var intervals int64
	for _, rn := range g.runs() {
		points := (rn.endMs-rn.startMs)/tickMs + 1
		if points > 1 {
			intervals += points - 1
		}
	}
	intervals *= int64(g.replicas)
	uptimeSeconds := float64(intervals) * tickSeconds

	const gib = 1024 * 1024 * 1024
	// CPU and egress close on exactly their per-day totals each active day (see
	// generate), so the billed total is the per-day amount times active days.
	cpuUsecPerDay := int64(g.vcpu * float64(g.intervalsPerDay()) * float64(tickMs) * 1000)
	cpuUsec := cpuUsecPerDay * int64(g.days) * int64(g.replicas)
	egressBytes := g.egressPerDay * int64(g.days) * int64(g.replicas)
	return billable{
		cpuSeconds:     float64(cpuUsec) / 1e6,
		memGiBSeconds:  float64(g.memoryBytes) / gib * uptimeSeconds,
		diskGiBSeconds: float64(g.diskBytes) / gib * uptimeSeconds,
		egressGiB:      float64(egressBytes) / gib,
	}
}
