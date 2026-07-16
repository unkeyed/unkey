// Package deployfail holds the user-facing failure messages the deploy worker
// writes to deployment_steps.error. The API read path matches on these exact
// constants to classify a failure into a stable DeploymentFailureCode. Sharing
// the strings here is what keeps the two sides from drifting: edit a message and
// both the producer and the classifier move together.
package deployfail

const (
	MsgNoSchedulableRegions = "No schedulable regions configured. Please configure at least one schedulable region before deploying."

	MsgCPUQuotaExceeded     = "We are unable to deploy this application as you have exceeded your CPU quota."
	MsgMemoryQuotaExceeded  = "We are unable to deploy this application as you have exceeded your Memory quota."
	MsgStorageQuotaExceeded = "We are unable to deploy this application as you have exceeded your Storage quota."

	MsgPortTooLow   = "Port must be greater than 0"
	MsgPortTooHigh  = "Port cannot exceed 65535"
	MsgCPUTooLow    = "CPU millicores must be greater than 0"
	MsgMemoryTooLow = "MemoryMib must be greater than 0"
)
