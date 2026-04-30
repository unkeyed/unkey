// Package checkpoint defines the unit heimdall writes to ClickHouse.
//
// A checkpoint is a single counter reading for one container. Billing math
// at query time is max(cpu_usage_usec) - min(cpu_usage_usec) over a window,
// which is monotone and replay-safe — the same checkpoint re-written cannot
// overcharge.
package checkpoint

import "strconv"

// Event kinds recorded on a checkpoint. Start/stop are emitted on CRI
// lifecycle events; periodic is the 10s safety-net reading.
const (
	EventStart    = "start"
	EventStop     = "stop"
	EventPeriodic = "checkpoint"
)

// ContainerUID returns a stable identifier for a single container lifecycle:
// the pod UID combined with the restart count. A restart produces a fresh
// cgroup with a fresh counter, so we never diff counters across UIDs.
func ContainerUID(podUID string, restartCount int32) string {
	return podUID + "/" + strconv.Itoa(int(restartCount))
}
