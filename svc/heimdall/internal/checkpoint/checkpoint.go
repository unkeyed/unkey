// Package checkpoint defines the unit heimdall writes to ClickHouse.
package checkpoint

import "strconv"

// Event kinds recorded on a checkpoint. Start/stop are emitted on CRI
// lifecycle events; periodic is the 10s safety-net reading.
const (
	EventStart    = "start"
	EventStop     = "stop"
	EventPeriodic = "checkpoint"
)

// ContainerUID combines the pod UID with the recorded restart count.
// Billing queries separate readings by this identifier even when the pod
// cgroup and its CPU counter persist across container restarts.
func ContainerUID(podUID string, restartCount int32) string {
	return podUID + "/" + strconv.Itoa(int(restartCount))
}
