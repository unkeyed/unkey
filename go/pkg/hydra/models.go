// Package hydra model types and constants
package hydra

import "github.com/unkeyed/unkey/go/pkg/hydra/store"

// WorkflowExecution represents a single workflow execution instance.
// It contains all metadata about a workflow run including status,
// timing, retry configuration, and trigger information.
type WorkflowExecution = store.WorkflowExecution

// WorkflowStep represents a single step within a workflow execution.
// Steps track their execution state, results, and retry attempts
// to enable checkpointing and exactly-once execution.
type WorkflowStep = store.WorkflowStep

// CronJob represents a scheduled workflow with cron-style timing.
// Cron jobs automatically create new workflow executions based
// on their schedule configuration.
type CronJob = store.CronJob

// Lease represents an exclusive lock on a workflow or step.
// Leases prevent multiple workers from executing the same
// work simultaneously and include heartbeat mechanisms.
type Lease = store.Lease

// WorkflowStatus represents the current state of a workflow execution.
type WorkflowStatus = store.WorkflowStatus

// StepStatus represents the current state of a workflow step.
type StepStatus = store.StepStatus

// LeaseKind identifies the type of resource being leased.
type LeaseKind = store.LeaseKind

// TriggerType identifies how a workflow execution was initiated.
type TriggerType = store.TriggerType

const (
	// Workflow status constants

	// WorkflowStatusPending indicates a workflow is waiting to be picked up by a worker.
	WorkflowStatusPending = store.WorkflowStatusPending

	// WorkflowStatusRunning indicates a workflow is currently being executed by a worker.
	WorkflowStatusRunning = store.WorkflowStatusRunning

	// WorkflowStatusSleeping indicates a workflow is suspended and waiting for a timer or external event.
	WorkflowStatusSleeping = store.WorkflowStatusSleeping

	// WorkflowStatusCompleted indicates a workflow has finished successfully.
	WorkflowStatusCompleted = store.WorkflowStatusCompleted

	// WorkflowStatusFailed indicates a workflow has failed and will not be retried.
	WorkflowStatusFailed = store.WorkflowStatusFailed

	// Step status constants

	// StepStatusPending indicates a step is waiting to be executed.
	StepStatusPending = store.StepStatusPending

	// StepStatusRunning indicates a step is currently being executed.
	StepStatusRunning = store.StepStatusRunning

	// StepStatusCompleted indicates a step has finished successfully and its result is cached.
	StepStatusCompleted = store.StepStatusCompleted

	// StepStatusFailed indicates a step has failed and may be retried.
	StepStatusFailed = store.StepStatusFailed

	// Lease kind constants

	// LeaseKindWorkflow indicates a lease on an entire workflow execution.
	LeaseKindWorkflow = store.LeaseKindWorkflow

	// LeaseKindStep indicates a lease on a specific workflow step.
	LeaseKindStep = store.LeaseKindStep

	// LeaseKindCronJob indicates a lease on a cron job execution.
	LeaseKindCronJob = store.LeaseKindCronJob

	// Trigger type constants

	// TriggerTypeManual indicates a workflow was started manually by an operator.
	TriggerTypeManual = store.TriggerTypeManual

	// TriggerTypeCron indicates a workflow was started by a cron schedule.
	TriggerTypeCron = store.TriggerTypeCron

	// TriggerTypeEvent indicates a workflow was started by an external event.
	TriggerTypeEvent = store.TriggerTypeEvent

	// TriggerTypeAPI indicates a workflow was started via the API.
	TriggerTypeAPI = store.TriggerTypeAPI
)

// RawPayload wraps raw byte data for workflow payloads.
// This type is used internally when the payload type is not known
// at compile time or when deserializing from the database.
type RawPayload struct {
	Data []byte `json:"data"`
}
