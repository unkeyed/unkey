package types

import (
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

type VMStatus int

const (
	VMStatusCreated VMStatus = iota
	VMStatusRunning
	VMStatusStopped
	VMStatusFailed
	VMStatusTerminated
)

var vmStatusStrings = map[VMStatus]string{
	VMStatusCreated:    "created",
	VMStatusRunning:    "running",
	VMStatusStopped:    "stopped",
	VMStatusFailed:     "failed",
	VMStatusTerminated: "terminated",
}

func (s VMStatus) String() string {
	if str, ok := vmStatusStrings[s]; ok {
		return str
	}

	return "unknown"
}

// ToProtoVmState converts our internal VMStatus to the protobuf VmState
func (s VMStatus) ToProtoVmState() metaldv1.VmState {
	switch s {
	case VMStatusCreated:
		return metaldv1.VmState_VM_STATE_CREATED
	case VMStatusRunning:
		return metaldv1.VmState_VM_STATE_RUNNING
	case VMStatusStopped, VMStatusTerminated:
		return metaldv1.VmState_VM_STATE_SHUTDOWN
	case VMStatusFailed:
		// Map failed to shutdown for now, as there's no failed state in proto
		return metaldv1.VmState_VM_STATE_SHUTDOWN
	default:
		return metaldv1.VmState_VM_STATE_UNSPECIFIED
	}
}

// VMStatusFromProto converts a protobuf VmState to our internal VMStatus
func VMStatusFromProto(state metaldv1.VmState) VMStatus {
	switch state {
	case metaldv1.VmState_VM_STATE_CREATED:
		return VMStatusCreated
	case metaldv1.VmState_VM_STATE_RUNNING:
		return VMStatusRunning
	case metaldv1.VmState_VM_STATE_PAUSED:
		return VMStatusStopped
	case metaldv1.VmState_VM_STATE_SHUTDOWN:
		return VMStatusTerminated
	default:
		return VMStatusCreated
	}
}
