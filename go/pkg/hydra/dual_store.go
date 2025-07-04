package hydra

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	sqlcstore "github.com/unkeyed/unkey/go/pkg/hydra/store/sqlc"
	"gorm.io/gorm"
)

// dualStore wraps both GORM and SQLC stores to enable incremental migration
type dualStore struct {
	gorm  store.Store        // GORM implementation (fallback)
	sqlc  *sqlcstore.Queries // SQLC implementation (migration target)
	db    *sql.DB            // Underlying database for transactions
	clock clock.Clock
}

// NewDualStore creates a store that can use both GORM and SQLC implementations
// It takes a GORM DB instance and creates a SQLC store from the same underlying connection
func NewDualStore(gormDB *gorm.DB, gormStore store.Store, clk clock.Clock) (store.Store, error) {
	if clk == nil {
		clk = clock.New()
	}

	// Get the underlying sql.DB from GORM
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	// Create SQLC queries
	sqlcQueries := sqlcstore.New(sqlDB)

	return &dualStore{
		gorm:  gormStore,
		sqlc:  sqlcQueries,
		db:    sqlDB,
		clock: clk,
	}, nil
}

// Workflow Execution Operations

func (s *dualStore) CreateWorkflow(ctx context.Context, workflow *store.WorkflowExecution) error {
	// TODO: Implement SQLC version
	return s.gorm.CreateWorkflow(ctx, workflow)
}

func (s *dualStore) GetWorkflow(ctx context.Context, namespace, id string) (*store.WorkflowExecution, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetWorkflow(ctx, namespace, id)
}

func (s *dualStore) GetPendingWorkflows(ctx context.Context, namespace string, limit int, workflowNames []string) ([]store.WorkflowExecution, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetPendingWorkflows(ctx, namespace, limit, workflowNames)
}

func (s *dualStore) GetPendingWorkflowsWithOffset(ctx context.Context, namespace string, limit int, offset int, workflowNames []string) ([]store.WorkflowExecution, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetPendingWorkflowsWithOffset(ctx, namespace, limit, offset, workflowNames)
}

func (s *dualStore) AcquireWorkflowLease(ctx context.Context, workflowID, namespace, workerID string, leaseDuration time.Duration) error {
	// TODO: Implement SQLC version with improved race condition handling
	return s.gorm.AcquireWorkflowLease(ctx, workflowID, namespace, workerID, leaseDuration)
}

func (s *dualStore) UpdateWorkflowStatus(ctx context.Context, namespace, id string, status store.WorkflowStatus, errorMsg string) error {
	// TODO: Implement SQLC version
	return s.gorm.UpdateWorkflowStatus(ctx, namespace, id, status, errorMsg)
}

func (s *dualStore) CompleteWorkflow(ctx context.Context, namespace, id string, outputData []byte) error {
	// TODO: Implement SQLC version
	return s.gorm.CompleteWorkflow(ctx, namespace, id, outputData)
}

func (s *dualStore) FailWorkflow(ctx context.Context, namespace, id string, errorMsg string, isFinal bool) error {
	// TODO: Implement SQLC version
	return s.gorm.FailWorkflow(ctx, namespace, id, errorMsg, isFinal)
}

func (s *dualStore) SleepWorkflow(ctx context.Context, namespace, id string, sleepUntil int64) error {
	// TODO: Implement SQLC version
	return s.gorm.SleepWorkflow(ctx, namespace, id, sleepUntil)
}

func (s *dualStore) GetSleepingWorkflows(ctx context.Context, namespace string, beforeTime int64) ([]store.WorkflowExecution, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetSleepingWorkflows(ctx, namespace, beforeTime)
}

func (s *dualStore) ResetOrphanedWorkflows(ctx context.Context, namespace string) error {
	// TODO: Implement SQLC version
	return s.gorm.ResetOrphanedWorkflows(ctx, namespace)
}

// Workflow Step Operations

func (s *dualStore) CreateStep(ctx context.Context, step *store.WorkflowStep) error {
	// TODO: Implement SQLC version
	return s.gorm.CreateStep(ctx, step)
}

func (s *dualStore) GetStep(ctx context.Context, namespace, executionID, stepName string) (*store.WorkflowStep, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetStep(ctx, namespace, executionID, stepName)
}

func (s *dualStore) GetCompletedStep(ctx context.Context, namespace, executionID, stepName string) (*store.WorkflowStep, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetCompletedStep(ctx, namespace, executionID, stepName)
}

func (s *dualStore) UpdateStepStatus(ctx context.Context, namespace, executionID, stepName string, status store.StepStatus, outputData []byte, errorMsg string) error {
	// TODO: Implement SQLC version
	return s.gorm.UpdateStepStatus(ctx, namespace, executionID, stepName, status, outputData, errorMsg)
}

// Lease Operations

func (s *dualStore) AcquireLease(ctx context.Context, lease *store.Lease) error {
	// TODO: Implement SQLC version
	return s.gorm.AcquireLease(ctx, lease)
}

func (s *dualStore) HeartbeatLease(ctx context.Context, resourceID, workerID string, expiresAt int64) error {
	// TODO: Implement SQLC version
	return s.gorm.HeartbeatLease(ctx, resourceID, workerID, expiresAt)
}

func (s *dualStore) ReleaseLease(ctx context.Context, resourceID, workerID string) error {
	// TODO: Implement SQLC version
	return s.gorm.ReleaseLease(ctx, resourceID, workerID)
}

func (s *dualStore) GetLease(ctx context.Context, resourceID string) (*store.Lease, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetLease(ctx, resourceID)
}

func (s *dualStore) CleanupExpiredLeases(ctx context.Context, namespace string) error {
	// TODO: Implement SQLC version
	return s.gorm.CleanupExpiredLeases(ctx, namespace)
}

func (s *dualStore) GetExpiredLeases(ctx context.Context, namespace string) ([]store.Lease, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetExpiredLeases(ctx, namespace)
}

// Cron Job Operations

func (s *dualStore) UpsertCronJob(ctx context.Context, cronJob *store.CronJob) error {
	// TODO: Implement SQLC version
	return s.gorm.UpsertCronJob(ctx, cronJob)
}

func (s *dualStore) GetCronJob(ctx context.Context, namespace, name string) (*store.CronJob, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetCronJob(ctx, namespace, name)
}

func (s *dualStore) GetCronJobs(ctx context.Context, namespace string) ([]store.CronJob, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetCronJobs(ctx, namespace)
}

func (s *dualStore) GetDueCronJobs(ctx context.Context, namespace string, beforeTime int64) ([]store.CronJob, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetDueCronJobs(ctx, namespace, beforeTime)
}

func (s *dualStore) UpdateCronJobLastRun(ctx context.Context, namespace, cronJobID string, lastRunAt, nextRunAt int64) error {
	// TODO: Implement SQLC version
	return s.gorm.UpdateCronJobLastRun(ctx, namespace, cronJobID, lastRunAt, nextRunAt)
}

// Transaction Support

func (s *dualStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	// TODO: Implement SQLC version with proper transaction handling
	return s.gorm.WithTx(ctx, fn)
}

// Testing helpers

func (s *dualStore) GetAllWorkflows(ctx context.Context, namespace string) ([]store.WorkflowExecution, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetAllWorkflows(ctx, namespace)
}

func (s *dualStore) GetAllSteps(ctx context.Context, namespace string) ([]store.WorkflowStep, error) {
	// TODO: Implement SQLC version
	return s.gorm.GetAllSteps(ctx, namespace)
}
