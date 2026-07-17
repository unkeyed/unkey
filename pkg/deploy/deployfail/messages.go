// Package deployfail holds the user-facing failure messages the deploy worker
// writes to deployment_steps.error. The API read path matches on these exact
// constants to classify a failure into a stable DeploymentFailureCode. Sharing
// the strings here is what keeps the two sides from drifting: edit a message and
// both the producer and the classifier move together.
package deployfail

const (
	MsgNoSchedulableRegions = "No regions configured. Please configure at least one region before deploying."

	MsgCPUQuotaExceeded     = "We are unable to deploy this application as you have exceeded your CPU quota."
	MsgMemoryQuotaExceeded  = "We are unable to deploy this application as you have exceeded your Memory quota."
	MsgStorageQuotaExceeded = "We are unable to deploy this application as you have exceeded your Storage quota."

	MsgPortTooLow   = "Port must be greater than 0"
	MsgPortTooHigh  = "Port cannot exceed 65535"
	MsgCPUTooLow    = "CPU millicores must be greater than 0"
	MsgMemoryTooLow = "MemoryMib must be greater than 0"
)

// RuntimeViolation is one runtime setting that fails a deploy precondition.
// Message is one of the Msg* constants above, so a violation stays classifiable
// by the read-path classifier; Actual is the offending value for reporting.
type RuntimeViolation struct {
	Message string
	Actual  int32
}

// RuntimeViolations reports which runtime settings would fail the deploy
// pipeline: port must be 1..65535, cpu and memory must be greater than 0. An
// empty result means the runtime settings are deployable. It is the single
// source of truth shared by the create-time gates (API, ctrl) and the worker.
func RuntimeViolations(port, cpuMillicores, memoryMib int32) []RuntimeViolation {
	var violations []RuntimeViolation
	switch {
	case port < 1:
		violations = append(violations, RuntimeViolation{Message: MsgPortTooLow, Actual: port})
	case port > 65535:
		violations = append(violations, RuntimeViolation{Message: MsgPortTooHigh, Actual: port})
	}
	if cpuMillicores < 1 {
		violations = append(violations, RuntimeViolation{Message: MsgCPUTooLow, Actual: cpuMillicores})
	}
	if memoryMib < 1 {
		violations = append(violations, RuntimeViolation{Message: MsgMemoryTooLow, Actual: memoryMib})
	}
	return violations
}
