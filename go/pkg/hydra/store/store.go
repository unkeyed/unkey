package store

import (
	"context"
	"time"
)

type Store interface {
	CreateWorkflow(ctx context.Context, workflow *WorkflowExecution) error

	GetWorkflow(ctx context.Context, namespace, id string) (*WorkflowExecution, error)

	GetPendingWorkflows(ctx context.Context, namespace string, limit int, workflowNames []string) ([]WorkflowExecution, error)

	AcquireWorkflowLease(ctx context.Context, workflowID, namespace, workerID string, leaseDuration time.Duration) error

	UpdateWorkflowStatus(ctx context.Context, namespace, id string, status WorkflowStatus, errorMsg string) error

	CompleteWorkflow(ctx context.Context, namespace, id string, outputData []byte) error

	FailWorkflow(ctx context.Context, namespace, id string, errorMsg string, isFinal bool) error

	SleepWorkflow(ctx context.Context, namespace, id string, sleepUntil int64) error

	GetSleepingWorkflows(ctx context.Context, namespace string, beforeTime int64) ([]WorkflowExecution, error)

	CreateStep(ctx context.Context, step *WorkflowStep) error

	GetStep(ctx context.Context, namespace, executionID, stepName string) (*WorkflowStep, error)

	GetCompletedStep(ctx context.Context, namespace, executionID, stepName string) (*WorkflowStep, error)

	UpdateStepStatus(ctx context.Context, namespace, executionID, stepName string, status StepStatus, outputData []byte, errorMsg string) error

	UpsertCronJob(ctx context.Context, cronJob *CronJob) error

	GetCronJob(ctx context.Context, namespace, name string) (*CronJob, error)

	GetCronJobs(ctx context.Context, namespace string) ([]CronJob, error)

	GetDueCronJobs(ctx context.Context, namespace string, beforeTime int64) ([]CronJob, error)

	UpdateCronJobLastRun(ctx context.Context, namespace, cronJobID string, lastRunAt, nextRunAt int64) error

	AcquireLease(ctx context.Context, lease *Lease) error

	HeartbeatLease(ctx context.Context, resourceID, workerID string, expiresAt int64) error

	ReleaseLease(ctx context.Context, resourceID, workerID string) error

	GetLease(ctx context.Context, resourceID string) (*Lease, error)

	CleanupExpiredLeases(ctx context.Context, namespace string) error

	GetExpiredLeases(ctx context.Context, namespace string) ([]Lease, error)

	ResetOrphanedWorkflows(ctx context.Context, namespace string) error

	WithTx(ctx context.Context, fn func(Store) error) error

	// Testing helpers
	GetAllWorkflows(ctx context.Context, namespace string) ([]WorkflowExecution, error)
	GetAllSteps(ctx context.Context, namespace string) ([]WorkflowStep, error)
}

type StoreFactory interface {
	NewStore() Store
}
