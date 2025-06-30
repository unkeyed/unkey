package store

type WorkflowExecution struct {
	ID           string `gorm:"primaryKey"`
	WorkflowName string
	Status       WorkflowStatus
	InputData    []byte
	OutputData   []byte
	ErrorMessage string

	CreatedAt   int64
	StartedAt   *int64
	CompletedAt *int64
	MaxAttempts       int32
	RemainingAttempts int32
	NextRetryAt       *int64

	Namespace string

	TriggerType   string
	TriggerSource *string

	SleepUntil *int64

	TraceID string
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}

type WorkflowStep struct {
	ID           string `gorm:"primaryKey"`
	ExecutionID  string
	StepName     string
	StepOrder    int32
	Status       StepStatus
	OutputData   []byte
	ErrorMessage string

	StartedAt   *int64
	CompletedAt *int64
	DurationMs  *int32

	MaxAttempts int32

	RemainingAttempts int32

	Namespace string
}

func (WorkflowStep) TableName() string {
	return "workflow_steps"
}

type CronJob struct {
	ID string `gorm:"primaryKey"`

	Name string

	CronSpec string

	Namespace string

	WorkflowName string

	Enabled bool

	CreatedAt int64

	UpdatedAt int64

	LastRunAt *int64

	NextRunAt int64
}

func (CronJob) TableName() string {
	return "cron_jobs"
}

type Lease struct {
	ResourceID string `gorm:"primaryKey"`

	Kind string

	Namespace string

	WorkerID string


	AcquiredAt int64

	ExpiresAt int64

	HeartbeatAt int64
}

func (Lease) TableName() string {
	return "leases"
}

type WorkflowStatus string

const (
	WorkflowStatusPending WorkflowStatus = "pending"

	WorkflowStatusRunning WorkflowStatus = "running"

	WorkflowStatusSleeping WorkflowStatus = "sleeping"

	WorkflowStatusCompleted WorkflowStatus = "completed"

	WorkflowStatusFailed WorkflowStatus = "failed"
)

type StepStatus string

const (
	StepStatusPending StepStatus = "pending"

	StepStatusRunning StepStatus = "running"

	StepStatusCompleted StepStatus = "completed"

	StepStatusFailed StepStatus = "failed"
)

type LeaseKind string

const (
	LeaseKindWorkflow LeaseKind = "workflow"

	LeaseKindStep LeaseKind = "step"

	LeaseKindCronJob LeaseKind = "cron_job"
)
