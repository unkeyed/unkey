// Package hydra model types and constants
package hydra

import "github.com/unkeyed/unkey/go/pkg/hydra/db"

// TriggerType identifies how a workflow execution was initiated.
type TriggerType = db.WorkflowExecutionsTriggerType

// Trigger type constants
const (
	// TriggerTypeManual indicates a workflow was started manually by an operator.
	TriggerTypeManual = db.WorkflowExecutionsTriggerTypeManual

	// TriggerTypeCron indicates a workflow was started by a cron schedule.
	TriggerTypeCron = db.WorkflowExecutionsTriggerTypeCron

	// TriggerTypeEvent indicates a workflow was started by an external event.
	TriggerTypeEvent = db.WorkflowExecutionsTriggerTypeEvent

	// TriggerTypeAPI indicates a workflow was started via the API.
	TriggerTypeAPI = db.WorkflowExecutionsTriggerTypeApi
)

// RawPayload wraps raw byte data for workflow payloads.
// This type is used internally when the payload type is not known
// at compile time or when deserializing from the database.
type RawPayload struct {
	Data []byte `json:"data"`
}
