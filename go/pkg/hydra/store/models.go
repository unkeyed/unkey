// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package store

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
)

type LeasesKind string

const (
	LeasesKindWorkflow LeasesKind = "workflow"
	LeasesKindStep     LeasesKind = "step"
	LeasesKindCronJob  LeasesKind = "cron_job"
)

func (e *LeasesKind) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = LeasesKind(s)
	case string:
		*e = LeasesKind(s)
	default:
		return fmt.Errorf("unsupported scan type for LeasesKind: %T", src)
	}
	return nil
}

type NullLeasesKind struct {
	LeasesKind LeasesKind `json:"leases_kind"`
	Valid      bool       `json:"valid"` // Valid is true if LeasesKind is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullLeasesKind) Scan(value interface{}) error {
	if value == nil {
		ns.LeasesKind, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.LeasesKind.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullLeasesKind) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.LeasesKind), nil
}

type WorkflowExecutionsStatus string

const (
	WorkflowExecutionsStatusPending   WorkflowExecutionsStatus = "pending"
	WorkflowExecutionsStatusRunning   WorkflowExecutionsStatus = "running"
	WorkflowExecutionsStatusSleeping  WorkflowExecutionsStatus = "sleeping"
	WorkflowExecutionsStatusCompleted WorkflowExecutionsStatus = "completed"
	WorkflowExecutionsStatusFailed    WorkflowExecutionsStatus = "failed"
)

func (e *WorkflowExecutionsStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = WorkflowExecutionsStatus(s)
	case string:
		*e = WorkflowExecutionsStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for WorkflowExecutionsStatus: %T", src)
	}
	return nil
}

type NullWorkflowExecutionsStatus struct {
	WorkflowExecutionsStatus WorkflowExecutionsStatus `json:"workflow_executions_status"`
	Valid                    bool                     `json:"valid"` // Valid is true if WorkflowExecutionsStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullWorkflowExecutionsStatus) Scan(value interface{}) error {
	if value == nil {
		ns.WorkflowExecutionsStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.WorkflowExecutionsStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullWorkflowExecutionsStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.WorkflowExecutionsStatus), nil
}

type WorkflowExecutionsTriggerType string

const (
	WorkflowExecutionsTriggerTypeManual WorkflowExecutionsTriggerType = "manual"
	WorkflowExecutionsTriggerTypeCron   WorkflowExecutionsTriggerType = "cron"
	WorkflowExecutionsTriggerTypeEvent  WorkflowExecutionsTriggerType = "event"
	WorkflowExecutionsTriggerTypeApi    WorkflowExecutionsTriggerType = "api"
)

func (e *WorkflowExecutionsTriggerType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = WorkflowExecutionsTriggerType(s)
	case string:
		*e = WorkflowExecutionsTriggerType(s)
	default:
		return fmt.Errorf("unsupported scan type for WorkflowExecutionsTriggerType: %T", src)
	}
	return nil
}

type NullWorkflowExecutionsTriggerType struct {
	WorkflowExecutionsTriggerType WorkflowExecutionsTriggerType `json:"workflow_executions_trigger_type"`
	Valid                         bool                          `json:"valid"` // Valid is true if WorkflowExecutionsTriggerType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullWorkflowExecutionsTriggerType) Scan(value interface{}) error {
	if value == nil {
		ns.WorkflowExecutionsTriggerType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.WorkflowExecutionsTriggerType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullWorkflowExecutionsTriggerType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.WorkflowExecutionsTriggerType), nil
}

type WorkflowStepsStatus string

const (
	WorkflowStepsStatusPending   WorkflowStepsStatus = "pending"
	WorkflowStepsStatusRunning   WorkflowStepsStatus = "running"
	WorkflowStepsStatusCompleted WorkflowStepsStatus = "completed"
	WorkflowStepsStatusFailed    WorkflowStepsStatus = "failed"
)

func (e *WorkflowStepsStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = WorkflowStepsStatus(s)
	case string:
		*e = WorkflowStepsStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for WorkflowStepsStatus: %T", src)
	}
	return nil
}

type NullWorkflowStepsStatus struct {
	WorkflowStepsStatus WorkflowStepsStatus `json:"workflow_steps_status"`
	Valid               bool                `json:"valid"` // Valid is true if WorkflowStepsStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullWorkflowStepsStatus) Scan(value interface{}) error {
	if value == nil {
		ns.WorkflowStepsStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.WorkflowStepsStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullWorkflowStepsStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.WorkflowStepsStatus), nil
}

type CronJob struct {
	ID           string         `db:"id" json:"id"`
	Name         string         `db:"name" json:"name"`
	CronSpec     string         `db:"cron_spec" json:"cron_spec"`
	Namespace    string         `db:"namespace" json:"namespace"`
	WorkflowName sql.NullString `db:"workflow_name" json:"workflow_name"`
	Enabled      bool           `db:"enabled" json:"enabled"`
	CreatedAt    int64          `db:"created_at" json:"created_at"`
	UpdatedAt    int64          `db:"updated_at" json:"updated_at"`
	LastRunAt    sql.NullInt64  `db:"last_run_at" json:"last_run_at"`
	NextRunAt    int64          `db:"next_run_at" json:"next_run_at"`
}

type Lease struct {
	ResourceID  string     `db:"resource_id" json:"resource_id"`
	Kind        LeasesKind `db:"kind" json:"kind"`
	Namespace   string     `db:"namespace" json:"namespace"`
	WorkerID    string     `db:"worker_id" json:"worker_id"`
	AcquiredAt  int64      `db:"acquired_at" json:"acquired_at"`
	ExpiresAt   int64      `db:"expires_at" json:"expires_at"`
	HeartbeatAt int64      `db:"heartbeat_at" json:"heartbeat_at"`
}

type WorkflowExecution struct {
	ID                string                            `db:"id" json:"id"`
	WorkflowName      string                            `db:"workflow_name" json:"workflow_name"`
	Status            WorkflowExecutionsStatus          `db:"status" json:"status"`
	InputData         []byte                            `db:"input_data" json:"input_data"`
	OutputData        []byte                            `db:"output_data" json:"output_data"`
	ErrorMessage      sql.NullString                    `db:"error_message" json:"error_message"`
	CreatedAt         int64                             `db:"created_at" json:"created_at"`
	StartedAt         sql.NullInt64                     `db:"started_at" json:"started_at"`
	CompletedAt       sql.NullInt64                     `db:"completed_at" json:"completed_at"`
	MaxAttempts       int32                             `db:"max_attempts" json:"max_attempts"`
	RemainingAttempts int32                             `db:"remaining_attempts" json:"remaining_attempts"`
	NextRetryAt       sql.NullInt64                     `db:"next_retry_at" json:"next_retry_at"`
	Namespace         string                            `db:"namespace" json:"namespace"`
	TriggerType       NullWorkflowExecutionsTriggerType `db:"trigger_type" json:"trigger_type"`
	TriggerSource     sql.NullString                    `db:"trigger_source" json:"trigger_source"`
	SleepUntil        sql.NullInt64                     `db:"sleep_until" json:"sleep_until"`
	TraceID           sql.NullString                    `db:"trace_id" json:"trace_id"`
	SpanID            sql.NullString                    `db:"span_id" json:"span_id"`
}

type WorkflowStep struct {
	ID                string              `db:"id" json:"id"`
	ExecutionID       string              `db:"execution_id" json:"execution_id"`
	StepName          string              `db:"step_name" json:"step_name"`
	Status            WorkflowStepsStatus `db:"status" json:"status"`
	OutputData        []byte              `db:"output_data" json:"output_data"`
	ErrorMessage      sql.NullString      `db:"error_message" json:"error_message"`
	StartedAt         sql.NullInt64       `db:"started_at" json:"started_at"`
	CompletedAt       sql.NullInt64       `db:"completed_at" json:"completed_at"`
	MaxAttempts       int32               `db:"max_attempts" json:"max_attempts"`
	RemainingAttempts int32               `db:"remaining_attempts" json:"remaining_attempts"`
	Namespace         string              `db:"namespace" json:"namespace"`
}
