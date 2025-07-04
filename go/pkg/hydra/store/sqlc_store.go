package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	sqlcstore "github.com/unkeyed/unkey/go/pkg/hydra/store/sqlc"
	"gorm.io/gorm"

	// MySQL driver for database/sql
	_ "github.com/go-sql-driver/mysql"
)

// sqlcStore implements the Store interface using SQLC instead of GORM
type sqlcStore struct {
	db      *sql.DB
	queries *sqlcstore.Queries
	clock   clock.Clock
}

// NewSQLCStore creates a new Store implementation using SQLC
// It takes the underlying sql.DB from a GORM instance to share the same connection
func NewSQLCStore(gormDB *gorm.DB, clk clock.Clock) (Store, error) {
	if clk == nil {
		clk = clock.New()
	}

	// Get the underlying sql.DB from GORM
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	queries := sqlcstore.New(sqlDB)

	return &sqlcStore{
		db:      sqlDB,
		queries: queries,
		clock:   clk,
	}, nil
}

// NewSQLCStoreFromDSN creates a new Store implementation using SQLC from a DSN
// This creates its own database connection independent of GORM
func NewSQLCStoreFromDSN(dsn string, clk clock.Clock) (Store, error) {
	if clk == nil {
		clk = clock.New()
	}

	// Open direct database connection for SQLC
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	queries := sqlcstore.New(sqlDB)

	return &sqlcStore{
		db:      sqlDB,
		queries: queries,
		clock:   clk,
	}, nil
}

// For now, just implement the one method we have a query for
func (s *sqlcStore) GetWorkflow(ctx context.Context, namespace, id string) (*WorkflowExecution, error) {
	workflow, err := s.queries.GetWorkflow(ctx, sqlcstore.GetWorkflowParams{
		ID:        id,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	// Convert SQLC model to store model
	result := &WorkflowExecution{
		ID:                workflow.ID,
		WorkflowName:      workflow.WorkflowName,
		Status:            WorkflowStatus(workflow.Status),
		InputData:         workflow.InputData,
		OutputData:        workflow.OutputData,
		CreatedAt:         workflow.CreatedAt,
		MaxAttempts:       workflow.MaxAttempts,
		RemainingAttempts: workflow.RemainingAttempts,
		Namespace:         workflow.Namespace,
	}

	// Handle nullable fields
	if workflow.ErrorMessage.Valid {
		result.ErrorMessage = workflow.ErrorMessage.String
	}
	if workflow.StartedAt.Valid {
		result.StartedAt = &workflow.StartedAt.Int64
	}
	if workflow.CompletedAt.Valid {
		result.CompletedAt = &workflow.CompletedAt.Int64
	}
	if workflow.NextRetryAt.Valid {
		result.NextRetryAt = &workflow.NextRetryAt.Int64
	}
	if workflow.TriggerType.Valid {
		result.TriggerType = TriggerType(workflow.TriggerType.WorkflowExecutionsTriggerType)
	}
	if workflow.TriggerSource.Valid {
		result.TriggerSource = &workflow.TriggerSource.String
	}
	if workflow.SleepUntil.Valid {
		result.SleepUntil = &workflow.SleepUntil.Int64
	}
	if workflow.TraceID.Valid {
		result.TraceID = workflow.TraceID.String
	}
	if workflow.SpanID.Valid {
		result.SpanID = workflow.SpanID.String
	}

	return result, nil
}

// Stub implementations for all other methods - will implement incrementally
func (s *sqlcStore) CreateWorkflow(ctx context.Context, workflow *WorkflowExecution) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetPendingWorkflows(ctx context.Context, namespace string, limit int, workflowNames []string) ([]WorkflowExecution, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) GetPendingWorkflowsWithOffset(ctx context.Context, namespace string, limit int, offset int, workflowNames []string) ([]WorkflowExecution, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) AcquireWorkflowLease(ctx context.Context, workflowID, namespace, workerID string, leaseDuration time.Duration) error {
	panic("not implemented yet")
}

func (s *sqlcStore) UpdateWorkflowStatus(ctx context.Context, namespace, id string, status WorkflowStatus, errorMsg string) error {
	panic("not implemented yet")
}

func (s *sqlcStore) CompleteWorkflow(ctx context.Context, namespace, id string, outputData []byte) error {
	panic("not implemented yet")
}

func (s *sqlcStore) FailWorkflow(ctx context.Context, namespace, id string, errorMsg string, isFinal bool) error {
	panic("not implemented yet")
}

func (s *sqlcStore) SleepWorkflow(ctx context.Context, namespace, id string, sleepUntil int64) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetSleepingWorkflows(ctx context.Context, namespace string, beforeTime int64) ([]WorkflowExecution, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) CreateStep(ctx context.Context, step *WorkflowStep) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetStep(ctx context.Context, namespace, executionID, stepName string) (*WorkflowStep, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) GetCompletedStep(ctx context.Context, namespace, executionID, stepName string) (*WorkflowStep, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) UpdateStepStatus(ctx context.Context, namespace, executionID, stepName string, status StepStatus, outputData []byte, errorMsg string) error {
	panic("not implemented yet")
}

func (s *sqlcStore) UpsertCronJob(ctx context.Context, cronJob *CronJob) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetCronJob(ctx context.Context, namespace, name string) (*CronJob, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) GetCronJobs(ctx context.Context, namespace string) ([]CronJob, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) GetDueCronJobs(ctx context.Context, namespace string, beforeTime int64) ([]CronJob, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) UpdateCronJobLastRun(ctx context.Context, namespace, cronJobID string, lastRunAt, nextRunAt int64) error {
	panic("not implemented yet")
}

func (s *sqlcStore) AcquireLease(ctx context.Context, lease *Lease) error {
	panic("not implemented yet")
}

func (s *sqlcStore) HeartbeatLease(ctx context.Context, resourceID, workerID string, expiresAt int64) error {
	panic("not implemented yet")
}

func (s *sqlcStore) ReleaseLease(ctx context.Context, resourceID, workerID string) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetLease(ctx context.Context, resourceID string) (*Lease, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) CleanupExpiredLeases(ctx context.Context, namespace string) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetExpiredLeases(ctx context.Context, namespace string) ([]Lease, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) ResetOrphanedWorkflows(ctx context.Context, namespace string) error {
	panic("not implemented yet")
}

func (s *sqlcStore) WithTx(ctx context.Context, fn func(Store) error) error {
	panic("not implemented yet")
}

func (s *sqlcStore) GetAllWorkflows(ctx context.Context, namespace string) ([]WorkflowExecution, error) {
	panic("not implemented yet")
}

func (s *sqlcStore) GetAllSteps(ctx context.Context, namespace string) ([]WorkflowStep, error) {
	panic("not implemented yet")
}
