package store

type WorkflowExecution struct {
	ID           string `gorm:"primaryKey"`
	WorkflowName string `gorm:"index:idx_workflow_namespace_name"`
	Status       WorkflowStatus `gorm:"index:idx_workflow_namespace_status;index:idx_workflow_status_retry;index:idx_workflow_status_sleep"`
	InputData    []byte
	OutputData   []byte
	ErrorMessage string

	CreatedAt         int64  `gorm:"index:idx_workflow_namespace_status"`
	StartedAt         *int64
	CompletedAt       *int64
	MaxAttempts       int32
	RemainingAttempts int32
	NextRetryAt       *int64 `gorm:"index:idx_workflow_status_retry"`

	Namespace string `gorm:"index:idx_workflow_namespace_status;index:idx_workflow_namespace_name;index:idx_workflow_status_retry;index:idx_workflow_status_sleep"`

	TriggerType   TriggerType
	TriggerSource *string

	SleepUntil *int64 `gorm:"index:idx_workflow_status_sleep"`

	TraceID string
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}

type WorkflowStep struct {
	ID           string `gorm:"primaryKey"`
	ExecutionID  string `gorm:"index:idx_workflow_step_unique,unique;index:idx_step_execution_status"`
	StepName     string `gorm:"index:idx_workflow_step_unique,unique"`
	StepOrder    int32
	Status       StepStatus `gorm:"index:idx_step_execution_status"`
	OutputData   []byte
	ErrorMessage string

	StartedAt   *int64
	CompletedAt *int64

	MaxAttempts int32

	RemainingAttempts int32

	Namespace string `gorm:"index:idx_workflow_step_unique,unique;index:idx_step_execution_status"`
}

func (WorkflowStep) TableName() string {
	return "workflow_steps"
}

type CronJob struct {
	ID string `gorm:"primaryKey"`

	Name string

	CronSpec string

	Namespace string `gorm:"index:idx_cron_namespace_enabled_next"`

	WorkflowName string

	Enabled bool `gorm:"index:idx_cron_namespace_enabled_next"`

	CreatedAt int64

	UpdatedAt int64

	LastRunAt *int64

	NextRunAt int64 `gorm:"index:idx_cron_namespace_enabled_next"`
}

func (CronJob) TableName() string {
	return "cron_jobs"
}

type Lease struct {
	ResourceID string `gorm:"primaryKey"`

	Kind string `gorm:"index:idx_lease_resource_kind"`

	Namespace string `gorm:"index:idx_lease_namespace_expires"`

	WorkerID string

	AcquiredAt int64

	ExpiresAt int64 `gorm:"index:idx_lease_namespace_expires"`

	HeartbeatAt int64
}

func (Lease) TableName() string {
	return "leases"
}

type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusSleeping  WorkflowStatus = "sleeping"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
)

// IsValid validates if the WorkflowStatus is one of the defined constants
func (ws WorkflowStatus) IsValid() bool {
	switch ws {
	case WorkflowStatusPending, WorkflowStatusRunning, WorkflowStatusSleeping, WorkflowStatusCompleted, WorkflowStatusFailed:
		return true
	default:
		return false
	}
}

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
)

// IsValid validates if the StepStatus is one of the defined constants
func (ss StepStatus) IsValid() bool {
	switch ss {
	case StepStatusPending, StepStatusRunning, StepStatusCompleted, StepStatusFailed:
		return true
	default:
		return false
	}
}

type LeaseKind string

const (
	LeaseKindWorkflow LeaseKind = "workflow"
	LeaseKindStep     LeaseKind = "step"
	LeaseKindCronJob  LeaseKind = "cron_job"
)

// IsValid validates if the LeaseKind is one of the defined constants
func (lk LeaseKind) IsValid() bool {
	switch lk {
	case LeaseKindWorkflow, LeaseKindStep, LeaseKindCronJob:
		return true
	default:
		return false
	}
}

type TriggerType string

const (
	TriggerTypeManual TriggerType = "manual"
	TriggerTypeCron   TriggerType = "cron"
	TriggerTypeEvent  TriggerType = "event"
	TriggerTypeAPI    TriggerType = "api"
)

// IsValid validates if the TriggerType is one of the defined constants
func (tt TriggerType) IsValid() bool {
	switch tt {
	case TriggerTypeManual, TriggerTypeCron, TriggerTypeEvent, TriggerTypeAPI:
		return true
	default:
		return false
	}
}
