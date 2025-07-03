package gorm

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrStepNotFound    = errors.New("step not found")
	ErrLeaseNotFound   = errors.New("lease not found")
	ErrCronJobNotFound = errors.New("cron job not found")

	// Empty structs for GORM Model() calls
	emptyWorkflowExecution = &store.WorkflowExecution{
		ID: "", WorkflowName: "", Status: "", InputData: nil, OutputData: nil,
		ErrorMessage: "", CreatedAt: 0, StartedAt: nil, CompletedAt: nil,
		MaxAttempts: 0, RemainingAttempts: 0, NextRetryAt: nil, Namespace: "",
		TriggerType: "", TriggerSource: nil, SleepUntil: nil, TraceID: "",
	}
	emptyWorkflowStep = &store.WorkflowStep{
		ID: "", ExecutionID: "", StepName: "", StepOrder: 0, Status: "",
		OutputData: nil, ErrorMessage: "", StartedAt: nil, CompletedAt: nil,
		MaxAttempts: 0, RemainingAttempts: 0, Namespace: "",
	}
	emptyCronJob = &store.CronJob{
		ID: "", Name: "", CronSpec: "", Namespace: "", WorkflowName: "",
		Enabled: false, CreatedAt: 0, UpdatedAt: 0, LastRunAt: nil, NextRunAt: 0,
	}
	emptyLease = &store.Lease{
		ResourceID: "", Kind: "", Namespace: "", WorkerID: "",
		AcquiredAt: 0, ExpiresAt: 0, HeartbeatAt: 0,
	}
)

type gormStore struct {
	db    *gorm.DB
	clock clock.Clock
}

func NewGORMStore(db *gorm.DB, clk clock.Clock) store.Store {
	if clk == nil {
		clk = clock.New() // Default to real clock
	}
	return &gormStore{db: db, clock: clk}
}

func NewSQLiteStore(dsn string, clk clock.Clock) (store.Store, error) {
	if dsn == "" {
		dsn = ":memory:" // Default to in-memory
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(emptyWorkflowExecution, emptyWorkflowStep, emptyCronJob, emptyLease); err != nil {
		return nil, err
	}

	return NewGORMStore(db, clk), nil
}

func NewMySQLStore(dsn string, clk clock.Clock) (store.Store, error) {
	if dsn == "" {
		return nil, errors.New("MySQL DSN is required")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(emptyWorkflowExecution, emptyWorkflowStep, emptyCronJob, emptyLease); err != nil {
		return nil, err
	}

	return NewGORMStore(db, clk), nil
}

func (s *gormStore) CreateWorkflow(ctx context.Context, workflow *store.WorkflowExecution) error {
	if workflow.CreatedAt == 0 {
		workflow.CreatedAt = s.clock.Now().UnixMilli()
	}
	return s.db.WithContext(ctx).Create(workflow).Error
}

func (s *gormStore) GetWorkflow(ctx context.Context, namespace, id string) (*store.WorkflowExecution, error) {
	var workflow store.WorkflowExecution
	err := s.db.WithContext(ctx).
		Where("id = ? AND namespace = ?", id, namespace).
		First(&workflow).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("workflow not found")
	}
	return &workflow, err
}

func (s *gormStore) GetPendingWorkflows(ctx context.Context, namespace string, limit int, workflowNames []string) ([]store.WorkflowExecution, error) {
	return s.GetPendingWorkflowsWithOffset(ctx, namespace, limit, 0, workflowNames)
}

func (s *gormStore) GetPendingWorkflowsWithOffset(ctx context.Context, namespace string, limit int, offset int, workflowNames []string) ([]store.WorkflowExecution, error) {
	now := time.Now().UnixMilli()
	var workflows []store.WorkflowExecution

	query := s.db.WithContext(ctx).
		Where("namespace = ? AND (status = ? OR (status = ? AND next_retry_at <= ?) OR (status = ? AND sleep_until <= ?))",
			namespace,
			store.WorkflowStatusPending,
			store.WorkflowStatusFailed,
			now,
			store.WorkflowStatusSleeping,
			now,
		)

	if len(workflowNames) > 0 {
		query = query.Where("workflow_name IN ?", workflowNames)
	}

	err := query.
		Order("created_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&workflows).Error

	return workflows, err
}

func (s *gormStore) AcquireWorkflowLease(ctx context.Context, workflowID, namespace, workerID string, leaseDuration time.Duration) error {
	now := time.Now().UnixMilli()
	expiresAt := now + leaseDuration.Milliseconds()

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, check if workflow is still available for leasing
		var workflow store.WorkflowExecution
		err := tx.Where("id = ? AND namespace = ?", workflowID, namespace).First(&workflow).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("workflow not found")
			}
			return err
		}

		// Check if workflow is in a leasable state
		if workflow.Status != store.WorkflowStatusPending &&
			workflow.Status != store.WorkflowStatusFailed &&
			workflow.Status != store.WorkflowStatusSleeping {
			return errors.New("workflow not available for acquisition")
		}

		// For failed workflows, check if retry time has passed
		if workflow.Status == store.WorkflowStatusFailed && workflow.NextRetryAt != nil && *workflow.NextRetryAt > now {
			return errors.New("workflow not ready for retry yet")
		}

		// For sleeping workflows, check if sleep time has passed
		if workflow.Status == store.WorkflowStatusSleeping && workflow.SleepUntil != nil && *workflow.SleepUntil > now {
			return errors.New("workflow still sleeping")
		}

		// Now check for existing lease
		var existingLease store.Lease
		err = tx.Where("resource_id = ? AND kind = ?", workflowID, "workflow").First(&existingLease).Error

		switch {
		case err == nil:
			if existingLease.ExpiresAt > now {
				if existingLease.WorkerID != workerID {
					return errors.New("workflow already leased by another worker")
				}
				// Renew existing lease
				existingLease.AcquiredAt = now
				existingLease.ExpiresAt = expiresAt
				existingLease.HeartbeatAt = now
				err = tx.Save(&existingLease).Error
				if err != nil {
					return err
				}
			} else {
				// Take over expired lease
				existingLease.WorkerID = workerID
				existingLease.AcquiredAt = now
				existingLease.ExpiresAt = expiresAt
				existingLease.HeartbeatAt = now
				err = tx.Save(&existingLease).Error
				if err != nil {
					return err
				}
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			// Create new lease
			lease := &store.Lease{
				ResourceID:  workflowID,
				Kind:        "workflow",
				Namespace:   namespace,
				WorkerID:    workerID,
				AcquiredAt:  now,
				ExpiresAt:   expiresAt,
				HeartbeatAt: now,
			}

			createErr := tx.Create(lease).Error
			if createErr != nil {
				if isDuplicateKeyError(createErr) {
					return errors.New("workflow already leased by another worker")
				}
				return createErr
			}
		default:
			return err
		}

		// Update workflow status to running
		result := tx.Model(emptyWorkflowExecution).
			Where("id = ? AND namespace = ?", workflowID, namespace).
			Updates(map[string]any{
				"status":      store.WorkflowStatusRunning,
				"started_at":  gorm.Expr("CASE WHEN started_at IS NULL THEN ? ELSE started_at END", now),
				"sleep_until": nil, // Clear sleep_until when waking up
			})

		if result.Error != nil {
			return result.Error
		}

		return nil
	})
}

func (s *gormStore) UpdateWorkflowStatus(ctx context.Context, namespace, id string, status store.WorkflowStatus, errorMsg string) error {
	updates := map[string]any{
		"status": status,
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return s.db.WithContext(ctx).
		Model(emptyWorkflowExecution).
		Where("id = ? AND namespace = ?", id, namespace).
		Updates(updates).Error
}

func (s *gormStore) CompleteWorkflow(ctx context.Context, namespace, id string, outputData []byte) error {
	now := time.Now().UnixMilli()
	updates := map[string]any{
		"status":       store.WorkflowStatusCompleted,
		"completed_at": now,
	}

	if outputData != nil {
		updates["output_data"] = outputData
	}

	return s.db.WithContext(ctx).
		Model(emptyWorkflowExecution).
		Where("id = ? AND namespace = ?", id, namespace).
		Updates(updates).Error
}

func (s *gormStore) FailWorkflow(ctx context.Context, namespace, id string, errorMsg string, isFinal bool) error {
	var workflow store.WorkflowExecution
	err := s.db.WithContext(ctx).
		Where("id = ? AND namespace = ?", id, namespace).
		First(&workflow).Error
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	workflow.ErrorMessage = errorMsg
	workflow.RemainingAttempts--

	updates := map[string]any{
		"error_message":      errorMsg,
		"remaining_attempts": workflow.RemainingAttempts,
	}

	if isFinal || workflow.RemainingAttempts <= 0 {
		updates["status"] = store.WorkflowStatusFailed
		updates["completed_at"] = now
		updates["next_retry_at"] = nil
	} else {
		updates["status"] = store.WorkflowStatusFailed
		attemptsUsed := workflow.MaxAttempts - workflow.RemainingAttempts
		backoffSeconds := int64(1 << attemptsUsed)
		nextRetry := now + (backoffSeconds * 1000)
		updates["next_retry_at"] = nextRetry
	}

	return s.db.WithContext(ctx).
		Model(emptyWorkflowExecution).
		Where("id = ? AND namespace = ?", id, namespace).
		Updates(updates).Error
}

func (s *gormStore) SleepWorkflow(ctx context.Context, namespace, id string, sleepUntil int64) error {
	updates := map[string]any{
		"status":      store.WorkflowStatusSleeping,
		"sleep_until": sleepUntil,
	}

	return s.db.WithContext(ctx).
		Model(emptyWorkflowExecution).
		Where("id = ? AND namespace = ?", id, namespace).
		Updates(updates).Error
}

func (s *gormStore) GetSleepingWorkflows(ctx context.Context, namespace string, beforeTime int64) ([]store.WorkflowExecution, error) {
	var workflows []store.WorkflowExecution

	err := s.db.WithContext(ctx).
		Where("namespace = ? AND status = ? AND sleep_until <= ?",
			namespace,
			store.WorkflowStatusSleeping,
			beforeTime,
		).
		Order("sleep_until ASC").
		Find(&workflows).Error

	return workflows, err
}

func (s *gormStore) CreateStep(ctx context.Context, step *store.WorkflowStep) error {
	if step.ID == "" {
		step.ID = step.ExecutionID + "-" + step.StepName
	}
	return s.db.WithContext(ctx).Create(step).Error
}

func (s *gormStore) GetStep(ctx context.Context, namespace, executionID, stepName string) (*store.WorkflowStep, error) {
	var step store.WorkflowStep
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND execution_id = ? AND step_name = ?",
			namespace, executionID, stepName).
		First(&step).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStepNotFound
	}
	return &step, err
}

func (s *gormStore) GetCompletedStep(ctx context.Context, namespace, executionID, stepName string) (*store.WorkflowStep, error) {
	var step store.WorkflowStep
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND execution_id = ? AND step_name = ? AND status = ?",
			namespace, executionID, stepName, store.StepStatusCompleted).
		First(&step).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStepNotFound
	}
	return &step, err
}

func (s *gormStore) UpdateStepStatus(ctx context.Context, namespace, executionID, stepName string, status store.StepStatus, outputData []byte, errorMsg string) error {
	now := time.Now().UnixMilli()

	var step store.WorkflowStep
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND execution_id = ? AND step_name = ?", namespace, executionID, stepName).
		First(&step).Error
	if err != nil {
		return err
	}

	updates := map[string]any{
		"status":       status,
		"completed_at": now,
	}

	if outputData != nil {
		updates["output_data"] = outputData
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return s.db.WithContext(ctx).
		Model(emptyWorkflowStep).
		Where("namespace = ? AND execution_id = ? AND step_name = ?", namespace, executionID, stepName).
		Updates(updates).Error
}

func (s *gormStore) AcquireLease(ctx context.Context, lease *store.Lease) error {
	result := s.db.WithContext(ctx).Create(lease)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return errors.New("lease already held by another worker")
		}
		return result.Error
	}
	return nil
}

func (s *gormStore) HeartbeatLease(ctx context.Context, resourceID, workerID string, expiresAt int64) error {
	now := time.Now().UnixMilli()

	result := s.db.WithContext(ctx).
		Model(emptyLease).
		Where("resource_id = ? AND worker_id = ?", resourceID, workerID).
		Updates(map[string]any{
			"heartbeat_at": now,
			"expires_at":   expiresAt,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("lease not found or not owned by worker")
	}

	return nil
}

func (s *gormStore) ReleaseLease(ctx context.Context, resourceID, workerID string) error {
	result := s.db.WithContext(ctx).
		Where("resource_id = ? AND worker_id = ?", resourceID, workerID).
		Delete(emptyLease)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("lease not found or not owned by worker")
	}

	return nil
}

func (s *gormStore) GetLease(ctx context.Context, resourceID string) (*store.Lease, error) {
	var lease store.Lease
	err := s.db.WithContext(ctx).
		Where("resource_id = ?", resourceID).
		First(&lease).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLeaseNotFound
	}

	return &lease, err
}

func (s *gormStore) CleanupExpiredLeases(ctx context.Context, namespace string) error {
	now := time.Now().UnixMilli()

	return s.db.WithContext(ctx).
		Where("namespace = ? AND expires_at < ?", namespace, now).
		Delete(emptyLease).Error
}

func (s *gormStore) GetExpiredLeases(ctx context.Context, namespace string) ([]store.Lease, error) {
	now := time.Now().UnixMilli()
	var leases []store.Lease

	err := s.db.WithContext(ctx).
		Where("namespace = ? AND expires_at < ?", namespace, now).
		Find(&leases).Error

	return leases, err
}

func (s *gormStore) ResetOrphanedWorkflows(ctx context.Context, namespace string) error {
	return s.db.WithContext(ctx).Exec(`
		UPDATE workflow_executions 
		SET status = 'pending' 
		WHERE namespace = ? 
		  AND status = 'running' 
		  AND id NOT IN (
		    SELECT resource_id 
		    FROM leases 
		    WHERE kind = 'workflow' AND namespace = ?
		  )
	`, namespace, namespace).Error
}

func (s *gormStore) UpsertCronJob(ctx context.Context, cronJob *store.CronJob) error {
	var existing store.CronJob
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND name = ?", cronJob.Namespace, cronJob.Name).
		First(&existing).Error

	if err == nil {
		cronJob.ID = existing.ID
		cronJob.CreatedAt = existing.CreatedAt
		cronJob.UpdatedAt = time.Now().UnixMilli()
		return s.db.WithContext(ctx).Save(cronJob).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.WithContext(ctx).Create(cronJob).Error
	}

	return err
}

func (s *gormStore) GetCronJob(ctx context.Context, namespace, name string) (*store.CronJob, error) {
	var cronJob store.CronJob
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND name = ?", namespace, name).
		First(&cronJob).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCronJobNotFound
	}

	return &cronJob, err
}

func (s *gormStore) GetCronJobs(ctx context.Context, namespace string) ([]store.CronJob, error) {
	var cronJobs []store.CronJob
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND enabled = ?", namespace, true).
		Find(&cronJobs).Error

	return cronJobs, err
}

func (s *gormStore) GetDueCronJobs(ctx context.Context, namespace string, beforeTime int64) ([]store.CronJob, error) {
	var cronJobs []store.CronJob
	err := s.db.WithContext(ctx).
		Where("namespace = ? AND enabled = ? AND next_run_at <= ?", namespace, true, beforeTime).
		Find(&cronJobs).Error

	return cronJobs, err
}

func (s *gormStore) UpdateCronJobLastRun(ctx context.Context, namespace, cronJobID string, lastRunAt, nextRunAt int64) error {
	return s.db.WithContext(ctx).
		Model(emptyCronJob).
		Where("id = ? AND namespace = ?", cronJobID, namespace).
		Updates(map[string]any{
			"last_run_at": lastRunAt,
			"next_run_at": nextRunAt,
			"updated_at":  time.Now().UnixMilli(),
		}).Error
}

func (s *gormStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txStore := &gormStore{db: tx, clock: s.clock}
		return fn(txStore)
	})
}

func isDuplicateKeyError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate") ||
		strings.Contains(errStr, "UNIQUE constraint") ||
		strings.Contains(errStr, "PRIMARY KEY constraint")
}

// Testing helpers
func (s *gormStore) GetAllWorkflows(ctx context.Context, namespace string) ([]store.WorkflowExecution, error) {
	var workflows []store.WorkflowExecution
	err := s.db.WithContext(ctx).
		Where("namespace = ?", namespace).
		Find(&workflows).Error
	return workflows, err
}

func (s *gormStore) GetAllSteps(ctx context.Context, namespace string) ([]store.WorkflowStep, error) {
	var steps []store.WorkflowStep
	err := s.db.WithContext(ctx).
		Where("namespace = ?", namespace).
		Find(&steps).Error
	return steps, err
}
