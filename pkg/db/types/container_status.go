package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ContainerStatus mirrors the kubelet-detailed shape we get from krane's
// pod-watch reports, denormalized onto the instances row. Stored in the
// `container_status` JSON column. Reads project the typed shape via the
// sqlc override; writes pass a typed value which the driver marshals to
// JSON via Value().
//
// Shape mirrors corev1.ContainerStatus:
//   - RestartCount: monotonic counter of kubelet-observed restarts. The
//     stale-event guards on UPDATE statements compare this against the
//     incoming counter to reject delayed RPCs from earlier container lives.
//   - LastTerminationState: most recent exit. Nil when the container has
//     never exited within retention.
//   - Waiting: current kubelet waiting reason (CrashLoopBackOff,
//     ImagePullBackOff, …). Nil when the container is running normally.
type ContainerStatus struct {
	RestartCount         uint32           `json:"restartCount"`
	LastTerminationState *TerminatedState `json:"lastTerminationState,omitempty"`
	Waiting              *WaitingState    `json:"waiting,omitempty"`
}

// TerminatedState carries the kubelet-supplied exit metadata for a
// finished container life. FinishedAt is unix milliseconds — same encoding
// as InstanceEvent.time on the wire so cross-table ORDER BY composes.
type TerminatedState struct {
	ExitCode   int32  `json:"exitCode"`
	Signal     int32  `json:"signal"`
	Reason     string `json:"reason"`
	FinishedAt int64  `json:"finishedAt"`
}

// WaitingState carries the kubelet-supplied waiting reason. The reason
// string is whatever kubelet publishes — krane currently only reports
// CrashLoopBackOff but the column can hold any future kind without a
// schema change.
type WaitingState struct {
	Reason string `json:"reason"`
}

// Scan implements [sql.Scanner] for reading the JSON column into a typed
// value. NULL columns return a zero-valued ContainerStatus, which matches
// the table default of {restartCount: 0}.
func (cs *ContainerStatus) Scan(value interface{}) error {
	if value == nil {
		*cs = ContainerStatus{
			RestartCount:         0,
			LastTerminationState: nil,
			Waiting:              nil,
		}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("ContainerStatus.Scan: expected []byte or string, got %T", value)
	}

	if len(bytes) == 0 || string(bytes) == "null" {
		*cs = ContainerStatus{
			RestartCount:         0,
			LastTerminationState: nil,
			Waiting:              nil,
		}
		return nil
	}

	return json.Unmarshal(bytes, cs)
}

// Value implements [driver.Valuer] for writing the typed value as JSON.
// Returning a string lets MySQL parse it into the JSON column directly;
// json.Marshal of this struct cannot fail (no channels, funcs, or cycles).
func (cs ContainerStatus) Value() (driver.Value, error) {
	bytes, err := json.Marshal(cs)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}
