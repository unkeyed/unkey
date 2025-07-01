package hydra

import "github.com/unkeyed/unkey/go/pkg/hydra/store"

type WorkflowExecution = store.WorkflowExecution

type WorkflowStep = store.WorkflowStep

type CronJob = store.CronJob

type Lease = store.Lease

type WorkflowStatus = store.WorkflowStatus

type StepStatus = store.StepStatus

type LeaseKind = store.LeaseKind

type TriggerType = store.TriggerType

const (
	WorkflowStatusPending   = store.WorkflowStatusPending
	WorkflowStatusRunning   = store.WorkflowStatusRunning
	WorkflowStatusSleeping  = store.WorkflowStatusSleeping
	WorkflowStatusCompleted = store.WorkflowStatusCompleted
	WorkflowStatusFailed    = store.WorkflowStatusFailed

	StepStatusPending   = store.StepStatusPending
	StepStatusRunning   = store.StepStatusRunning
	StepStatusCompleted = store.StepStatusCompleted
	StepStatusFailed    = store.StepStatusFailed

	LeaseKindWorkflow = store.LeaseKindWorkflow
	LeaseKindStep     = store.LeaseKindStep
	LeaseKindCronJob  = store.LeaseKindCronJob

	TriggerTypeManual = store.TriggerTypeManual
	TriggerTypeCron   = store.TriggerTypeCron
	TriggerTypeEvent  = store.TriggerTypeEvent
	TriggerTypeAPI    = store.TriggerTypeAPI
)

type RawPayload struct {
	Data []byte `json:"data"`
}
