package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	sqlcstore "github.com/unkeyed/unkey/go/pkg/hydra/store/sqlc"

	// MySQL driver for database/sql
	_ "github.com/go-sql-driver/mysql"
)

var (
	ErrStepNotFound = errors.New("step not found")
)

// sqlcStore implements the Store interface using SQLC instead of GORM
type sqlcStore struct {
	db      *sql.DB
	queries *sqlcstore.Queries
	clock   clock.Clock
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

// CreateWorkflow creates a new workflow execution record
func (s *sqlcStore) CreateWorkflow(ctx context.Context, workflow *WorkflowExecution) error {
	// Set CreatedAt if not already set
	if workflow.CreatedAt == 0 {
		workflow.CreatedAt = s.clock.Now().UnixMilli()
	}

	// Convert store model to SQLC parameters
	var startedAt, completedAt, nextRetryAt, sleepUntil sql.NullInt64
	var triggerSource sql.NullString

	if workflow.StartedAt != nil {
		startedAt = sql.NullInt64{Int64: *workflow.StartedAt, Valid: true}
	}
	if workflow.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: *workflow.CompletedAt, Valid: true}
	}
	if workflow.NextRetryAt != nil {
		nextRetryAt = sql.NullInt64{Int64: *workflow.NextRetryAt, Valid: true}
	}
	if workflow.SleepUntil != nil {
		sleepUntil = sql.NullInt64{Int64: *workflow.SleepUntil, Valid: true}
	}
	if workflow.TriggerSource != nil {
		triggerSource = sql.NullString{String: *workflow.TriggerSource, Valid: true}
	}

	params := sqlcstore.CreateWorkflowParams{
		ID:                workflow.ID,
		WorkflowName:      workflow.WorkflowName,
		Status:            sqlcstore.WorkflowExecutionsStatus(workflow.Status),
		InputData:         workflow.InputData,
		OutputData:        workflow.OutputData,
		ErrorMessage:      sql.NullString{String: workflow.ErrorMessage, Valid: workflow.ErrorMessage != ""},
		CreatedAt:         workflow.CreatedAt,
		StartedAt:         startedAt,
		CompletedAt:       completedAt,
		MaxAttempts:       int32(workflow.MaxAttempts),
		RemainingAttempts: int32(workflow.RemainingAttempts),
		NextRetryAt:       nextRetryAt,
		Namespace:         workflow.Namespace,
		TriggerType:       sqlcstore.NullWorkflowExecutionsTriggerType{WorkflowExecutionsTriggerType: sqlcstore.WorkflowExecutionsTriggerType(workflow.TriggerType), Valid: workflow.TriggerType != ""},
		TriggerSource:     triggerSource,
		SleepUntil:        sleepUntil,
		TraceID:           sql.NullString{String: workflow.TraceID, Valid: workflow.TraceID != ""},
		SpanID:            sql.NullString{String: workflow.SpanID, Valid: workflow.SpanID != ""},
	}

	return s.queries.CreateWorkflow(ctx, params)
}

func (s *sqlcStore) GetPendingWorkflows(ctx context.Context, namespace string, limit int, workflowNames []string) ([]WorkflowExecution, error) {
	now := s.clock.Now().UnixMilli()

	// If workflowNames filter is empty, get all pending workflows
	if len(workflowNames) == 0 {
		params := sqlcstore.GetPendingWorkflowsParams{
			Namespace:   namespace,
			NextRetryAt: sql.NullInt64{Int64: now, Valid: true},
			SleepUntil:  sql.NullInt64{Int64: now, Valid: true},
			Limit:       int32(limit),
		}

		sqlcWorkflows, err := s.queries.GetPendingWorkflows(ctx, params)
		if err != nil {
			return nil, err
		}

		// Convert SQLC models to store models
		result := make([]WorkflowExecution, len(sqlcWorkflows))
		for i, sqlcWf := range sqlcWorkflows {
			result[i] = WorkflowExecution{
				ID:                sqlcWf.ID,
				WorkflowName:      sqlcWf.WorkflowName,
				Status:            WorkflowStatus(sqlcWf.Status),
				InputData:         sqlcWf.InputData,
				OutputData:        sqlcWf.OutputData,
				ErrorMessage:      sqlcWf.ErrorMessage.String,
				CreatedAt:         sqlcWf.CreatedAt,
				StartedAt:         nullInt64ToPtr(sqlcWf.StartedAt),
				CompletedAt:       nullInt64ToPtr(sqlcWf.CompletedAt),
				MaxAttempts:       sqlcWf.MaxAttempts,
				RemainingAttempts: sqlcWf.RemainingAttempts,
				NextRetryAt:       nullInt64ToPtr(sqlcWf.NextRetryAt),
				Namespace:         sqlcWf.Namespace,
				TriggerType:       TriggerType(sqlcWf.TriggerType.WorkflowExecutionsTriggerType),
				TriggerSource:     nullStringToPtr(sqlcWf.TriggerSource),
				SleepUntil:        nullInt64ToPtr(sqlcWf.SleepUntil),
				TraceID:           sqlcWf.TraceID.String,
				SpanID:            sqlcWf.SpanID.String,
			}
		}
		return result, nil
	}

	// For filtered queries, we need to call the query for each workflow name
	// This is less efficient but handles the slice limitation
	var allWorkflows []WorkflowExecution
	remainingLimit := limit

	for _, workflowName := range workflowNames {
		if remainingLimit <= 0 {
			break
		}

		params := sqlcstore.GetPendingWorkflowsFilteredParams{
			Namespace:    namespace,
			NextRetryAt:  sql.NullInt64{Int64: now, Valid: true},
			SleepUntil:   sql.NullInt64{Int64: now, Valid: true},
			WorkflowName: workflowName,
			Limit:        int32(remainingLimit),
		}

		sqlcWorkflows, err := s.queries.GetPendingWorkflowsFiltered(ctx, params)
		if err != nil {
			return nil, err
		}

		// Convert and append
		for _, sqlcWf := range sqlcWorkflows {
			if len(allWorkflows) >= limit {
				break
			}
			allWorkflows = append(allWorkflows, WorkflowExecution{
				ID:                sqlcWf.ID,
				WorkflowName:      sqlcWf.WorkflowName,
				Status:            WorkflowStatus(sqlcWf.Status),
				InputData:         sqlcWf.InputData,
				OutputData:        sqlcWf.OutputData,
				ErrorMessage:      sqlcWf.ErrorMessage.String,
				CreatedAt:         sqlcWf.CreatedAt,
				StartedAt:         nullInt64ToPtr(sqlcWf.StartedAt),
				CompletedAt:       nullInt64ToPtr(sqlcWf.CompletedAt),
				MaxAttempts:       sqlcWf.MaxAttempts,
				RemainingAttempts: sqlcWf.RemainingAttempts,
				NextRetryAt:       nullInt64ToPtr(sqlcWf.NextRetryAt),
				Namespace:         sqlcWf.Namespace,
				TriggerType:       TriggerType(sqlcWf.TriggerType.WorkflowExecutionsTriggerType),
				TriggerSource:     nullStringToPtr(sqlcWf.TriggerSource),
				SleepUntil:        nullInt64ToPtr(sqlcWf.SleepUntil),
				TraceID:           sqlcWf.TraceID.String,
				SpanID:            sqlcWf.SpanID.String,
			})
		}

		remainingLimit = limit - len(allWorkflows)
	}

	return allWorkflows, nil
}

func (s *sqlcStore) AcquireWorkflowLease(ctx context.Context, workflowID, namespace, workerID string, leaseDuration time.Duration) error {
	now := s.clock.Now().UnixMilli()
	expiresAt := now + leaseDuration.Milliseconds()

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create queries instance for the transaction
	txQueries := sqlcstore.New(tx)

	// First, check if workflow is still available for leasing
	workflow, err := txQueries.GetWorkflow(ctx, sqlcstore.GetWorkflowParams{
		ID:        workflowID,
		Namespace: namespace,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("workflow not found")
		}
		return err
	}

	// Check if workflow is in a leasable state
	status := WorkflowStatus(workflow.Status)
	if status != WorkflowStatusPending &&
		status != WorkflowStatusFailed &&
		status != WorkflowStatusSleeping {
		return errors.New("workflow not available for acquisition")
	}

	// For failed workflows, check if retry time has passed
	if status == WorkflowStatusFailed && workflow.NextRetryAt.Valid && workflow.NextRetryAt.Int64 > now {
		return errors.New("workflow not ready for retry yet")
	}

	// For sleeping workflows, check if sleep time has passed
	if status == WorkflowStatusSleeping && workflow.SleepUntil.Valid && workflow.SleepUntil.Int64 > now {
		return errors.New("workflow still sleeping")
	}

	// Check for existing lease
	existingLease, err := txQueries.GetLease(ctx, sqlcstore.GetLeaseParams{
		ResourceID: workflowID,
		Kind:       "workflow",
	})

	switch {
	case err == nil:
		// Lease exists
		if existingLease.ExpiresAt > now {
			if existingLease.WorkerID != workerID {
				return errors.New("workflow already leased by another worker")
			}
			// Renew existing lease
			err = txQueries.UpdateLease(ctx, sqlcstore.UpdateLeaseParams{
				WorkerID:    workerID,
				AcquiredAt:  now,
				ExpiresAt:   expiresAt,
				HeartbeatAt: now,
				ResourceID:  workflowID,
				Kind:        "workflow",
			})
			if err != nil {
				return err
			}
		} else {
			// Take over expired lease
			err = txQueries.UpdateLease(ctx, sqlcstore.UpdateLeaseParams{
				WorkerID:    workerID,
				AcquiredAt:  now,
				ExpiresAt:   expiresAt,
				HeartbeatAt: now,
				ResourceID:  workflowID,
				Kind:        "workflow",
			})
			if err != nil {
				return err
			}
		}
	case err == sql.ErrNoRows:
		// Create new lease
		err = txQueries.CreateLease(ctx, sqlcstore.CreateLeaseParams{
			ResourceID:  workflowID,
			Kind:        "workflow",
			Namespace:   namespace,
			WorkerID:    workerID,
			AcquiredAt:  now,
			ExpiresAt:   expiresAt,
			HeartbeatAt: now,
		})
		if err != nil {
			// Check if it's a duplicate key error (race condition)
			if isDuplicateKeyError(err) {
				return errors.New("workflow already leased by another worker")
			}
			return err
		}
	default:
		return err
	}

	// Update workflow status to running
	err = txQueries.UpdateWorkflowToRunning(ctx, sqlcstore.UpdateWorkflowToRunningParams{
		StartedAt: sql.NullInt64{Int64: now, Valid: true},
		ID:        workflowID,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}

func (s *sqlcStore) UpdateWorkflowStatus(ctx context.Context, namespace, id string, status WorkflowStatus, errorMsg string) error {
	params := sqlcstore.UpdateWorkflowStatusParams{
		Status:       sqlcstore.WorkflowExecutionsStatus(status),
		ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		ID:           id,
		Namespace:    namespace,
	}

	return s.queries.UpdateWorkflowStatus(ctx, params)
}

func (s *sqlcStore) CompleteWorkflow(ctx context.Context, namespace, id string, outputData []byte) error {
	now := s.clock.Now().UnixMilli()

	params := sqlcstore.CompleteWorkflowParams{
		CompletedAt: sql.NullInt64{Int64: now, Valid: true},
		OutputData:  outputData,
		ID:          id,
		Namespace:   namespace,
	}

	return s.queries.CompleteWorkflow(ctx, params)
}

func (s *sqlcStore) FailWorkflow(ctx context.Context, namespace, id string, errorMsg string, isFinal bool) error {
	// First get the current workflow to check remaining attempts
	workflow, err := s.GetWorkflow(ctx, namespace, id)
	if err != nil {
		return err
	}

	now := s.clock.Now().UnixMilli()
	newRemainingAttempts := workflow.RemainingAttempts - 1

	if isFinal || newRemainingAttempts <= 0 {
		// Final failure - no more retries
		params := sqlcstore.FailWorkflowFinalParams{
			ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
			CompletedAt:  sql.NullInt64{Int64: now, Valid: true},
			ID:           id,
			Namespace:    namespace,
		}
		return s.queries.FailWorkflowFinal(ctx, params)
	} else {
		// Retry-eligible failure - calculate next retry time
		attemptsUsed := workflow.MaxAttempts - newRemainingAttempts
		backoffSeconds := int64(1 << attemptsUsed)
		nextRetry := now + (backoffSeconds * 1000)

		params := sqlcstore.FailWorkflowWithRetryParams{
			ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
			NextRetryAt:  sql.NullInt64{Int64: nextRetry, Valid: true},
			ID:           id,
			Namespace:    namespace,
		}
		return s.queries.FailWorkflowWithRetry(ctx, params)
	}
}

func (s *sqlcStore) SleepWorkflow(ctx context.Context, namespace, id string, sleepUntil int64) error {
	params := sqlcstore.SleepWorkflowParams{
		SleepUntil: sql.NullInt64{Int64: sleepUntil, Valid: true},
		ID:         id,
		Namespace:  namespace,
	}

	return s.queries.SleepWorkflow(ctx, params)
}

func (s *sqlcStore) GetSleepingWorkflows(ctx context.Context, namespace string, beforeTime int64) ([]WorkflowExecution, error) {
	params := sqlcstore.GetSleepingWorkflowsParams{
		Namespace:  namespace,
		SleepUntil: sql.NullInt64{Int64: beforeTime, Valid: true},
	}

	sqlcWorkflows, err := s.queries.GetSleepingWorkflows(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert SQLC models to store models
	result := make([]WorkflowExecution, len(sqlcWorkflows))
	for i, sqlcWf := range sqlcWorkflows {
		result[i] = WorkflowExecution{
			ID:                sqlcWf.ID,
			WorkflowName:      sqlcWf.WorkflowName,
			Status:            WorkflowStatus(sqlcWf.Status),
			InputData:         sqlcWf.InputData,
			OutputData:        sqlcWf.OutputData,
			ErrorMessage:      sqlcWf.ErrorMessage.String,
			CreatedAt:         sqlcWf.CreatedAt,
			StartedAt:         nullInt64ToPtr(sqlcWf.StartedAt),
			CompletedAt:       nullInt64ToPtr(sqlcWf.CompletedAt),
			MaxAttempts:       sqlcWf.MaxAttempts,
			RemainingAttempts: sqlcWf.RemainingAttempts,
			NextRetryAt:       nullInt64ToPtr(sqlcWf.NextRetryAt),
			Namespace:         sqlcWf.Namespace,
			TriggerType:       TriggerType(sqlcWf.TriggerType.WorkflowExecutionsTriggerType),
			TriggerSource:     nullStringToPtr(sqlcWf.TriggerSource),
			SleepUntil:        nullInt64ToPtr(sqlcWf.SleepUntil),
			TraceID:           sqlcWf.TraceID.String,
			SpanID:            sqlcWf.SpanID.String,
		}
	}
	return result, nil
}

func (s *sqlcStore) CreateStep(ctx context.Context, step *WorkflowStep) error {
	// Set ID if not already set (following GORM logic)
	if step.ID == "" {
		step.ID = step.ExecutionID + "-" + step.StepName
	}

	var startedAt, completedAt sql.NullInt64
	if step.StartedAt != nil {
		startedAt = sql.NullInt64{Int64: *step.StartedAt, Valid: true}
	}
	if step.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: *step.CompletedAt, Valid: true}
	}

	params := sqlcstore.CreateStepParams{
		ID:                step.ID,
		ExecutionID:       step.ExecutionID,
		StepName:          step.StepName,
		StepOrder:         step.StepOrder,
		Status:            sqlcstore.WorkflowStepsStatus(step.Status),
		OutputData:        step.OutputData,
		ErrorMessage:      sql.NullString{String: step.ErrorMessage, Valid: step.ErrorMessage != ""},
		StartedAt:         startedAt,
		CompletedAt:       completedAt,
		MaxAttempts:       step.MaxAttempts,
		RemainingAttempts: step.RemainingAttempts,
		Namespace:         step.Namespace,
	}

	return s.queries.CreateStep(ctx, params)
}

func (s *sqlcStore) GetStep(ctx context.Context, namespace, executionID, stepName string) (*WorkflowStep, error) {
	params := sqlcstore.GetStepParams{
		Namespace:   namespace,
		ExecutionID: executionID,
		StepName:    stepName,
	}

	sqlcStep, err := s.queries.GetStep(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrStepNotFound
		}
		return nil, err
	}

	// Convert SQLC model to store model
	result := &WorkflowStep{
		ID:                sqlcStep.ID,
		ExecutionID:       sqlcStep.ExecutionID,
		StepName:          sqlcStep.StepName,
		StepOrder:         sqlcStep.StepOrder,
		Status:            StepStatus(sqlcStep.Status),
		OutputData:        sqlcStep.OutputData,
		ErrorMessage:      sqlcStep.ErrorMessage.String,
		StartedAt:         nullInt64ToPtr(sqlcStep.StartedAt),
		CompletedAt:       nullInt64ToPtr(sqlcStep.CompletedAt),
		MaxAttempts:       sqlcStep.MaxAttempts,
		RemainingAttempts: sqlcStep.RemainingAttempts,
		Namespace:         sqlcStep.Namespace,
	}

	return result, nil
}

func (s *sqlcStore) GetCompletedStep(ctx context.Context, namespace, executionID, stepName string) (*WorkflowStep, error) {
	params := sqlcstore.GetCompletedStepParams{
		Namespace:   namespace,
		ExecutionID: executionID,
		StepName:    stepName,
	}

	sqlcStep, err := s.queries.GetCompletedStep(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrStepNotFound
		}
		return nil, err
	}

	// Convert SQLC model to store model
	result := &WorkflowStep{
		ID:                sqlcStep.ID,
		ExecutionID:       sqlcStep.ExecutionID,
		StepName:          sqlcStep.StepName,
		StepOrder:         sqlcStep.StepOrder,
		Status:            StepStatus(sqlcStep.Status),
		OutputData:        sqlcStep.OutputData,
		ErrorMessage:      sqlcStep.ErrorMessage.String,
		StartedAt:         nullInt64ToPtr(sqlcStep.StartedAt),
		CompletedAt:       nullInt64ToPtr(sqlcStep.CompletedAt),
		MaxAttempts:       sqlcStep.MaxAttempts,
		RemainingAttempts: sqlcStep.RemainingAttempts,
		Namespace:         sqlcStep.Namespace,
	}

	return result, nil
}

func (s *sqlcStore) UpdateStepStatus(ctx context.Context, namespace, executionID, stepName string, status StepStatus, outputData []byte, errorMsg string) error {
	now := s.clock.Now().UnixMilli()

	params := sqlcstore.UpdateStepStatusParams{
		Status:       sqlcstore.WorkflowStepsStatus(status),
		CompletedAt:  sql.NullInt64{Int64: now, Valid: true},
		OutputData:   outputData,
		ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		Namespace:    namespace,
		ExecutionID:  executionID,
		StepName:     stepName,
	}

	return s.queries.UpdateStepStatus(ctx, params)
}

func (s *sqlcStore) UpsertCronJob(ctx context.Context, cronJob *CronJob) error {
	// First try to get existing cron job
	existing, err := s.queries.GetCronJob(ctx, sqlcstore.GetCronJobParams{
		Namespace: cronJob.Namespace,
		Name:      cronJob.Name,
	})

	if err == sql.ErrNoRows {
		// Create new cron job
		var lastRunAt sql.NullInt64
		if cronJob.LastRunAt != nil {
			lastRunAt = sql.NullInt64{Int64: *cronJob.LastRunAt, Valid: true}
		}

		params := sqlcstore.CreateCronJobParams{
			ID:           cronJob.ID,
			Name:         cronJob.Name,
			CronSpec:     cronJob.CronSpec,
			Namespace:    cronJob.Namespace,
			WorkflowName: sql.NullString{String: cronJob.WorkflowName, Valid: cronJob.WorkflowName != ""},
			Enabled:      cronJob.Enabled,
			CreatedAt:    cronJob.CreatedAt,
			UpdatedAt:    cronJob.UpdatedAt,
			LastRunAt:    lastRunAt,
			NextRunAt:    cronJob.NextRunAt,
		}
		return s.queries.CreateCronJob(ctx, params)
	} else if err != nil {
		return err
	}

	// Update existing cron job
	cronJob.ID = existing.ID
	cronJob.CreatedAt = existing.CreatedAt
	cronJob.UpdatedAt = s.clock.Now().UnixMilli()

	params := sqlcstore.UpdateCronJobParams{
		CronSpec:     cronJob.CronSpec,
		WorkflowName: sql.NullString{String: cronJob.WorkflowName, Valid: cronJob.WorkflowName != ""},
		Enabled:      cronJob.Enabled,
		UpdatedAt:    cronJob.UpdatedAt,
		NextRunAt:    cronJob.NextRunAt,
		ID:           cronJob.ID,
		Namespace:    cronJob.Namespace,
	}
	return s.queries.UpdateCronJob(ctx, params)
}

func (s *sqlcStore) GetCronJob(ctx context.Context, namespace, name string) (*CronJob, error) {
	params := sqlcstore.GetCronJobParams{
		Namespace: namespace,
		Name:      name,
	}

	sqlcCronJob, err := s.queries.GetCronJob(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("cron job not found")
		}
		return nil, err
	}

	// Convert SQLC model to store model
	result := &CronJob{
		ID:           sqlcCronJob.ID,
		Name:         sqlcCronJob.Name,
		CronSpec:     sqlcCronJob.CronSpec,
		Namespace:    sqlcCronJob.Namespace,
		WorkflowName: sqlcCronJob.WorkflowName.String,
		Enabled:      sqlcCronJob.Enabled,
		CreatedAt:    sqlcCronJob.CreatedAt,
		UpdatedAt:    sqlcCronJob.UpdatedAt,
		LastRunAt:    nullInt64ToPtr(sqlcCronJob.LastRunAt),
		NextRunAt:    sqlcCronJob.NextRunAt,
	}

	return result, nil
}

func (s *sqlcStore) GetCronJobs(ctx context.Context, namespace string) ([]CronJob, error) {
	sqlcCronJobs, err := s.queries.GetCronJobs(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// Convert SQLC models to store models
	result := make([]CronJob, len(sqlcCronJobs))
	for i, sqlcCron := range sqlcCronJobs {
		result[i] = CronJob{
			ID:           sqlcCron.ID,
			Name:         sqlcCron.Name,
			CronSpec:     sqlcCron.CronSpec,
			Namespace:    sqlcCron.Namespace,
			WorkflowName: sqlcCron.WorkflowName.String,
			Enabled:      sqlcCron.Enabled,
			CreatedAt:    sqlcCron.CreatedAt,
			UpdatedAt:    sqlcCron.UpdatedAt,
			LastRunAt:    nullInt64ToPtr(sqlcCron.LastRunAt),
			NextRunAt:    sqlcCron.NextRunAt,
		}
	}
	return result, nil
}

func (s *sqlcStore) GetDueCronJobs(ctx context.Context, namespace string, beforeTime int64) ([]CronJob, error) {
	params := sqlcstore.GetDueCronJobsParams{
		Namespace: namespace,
		NextRunAt: beforeTime,
	}

	sqlcCronJobs, err := s.queries.GetDueCronJobs(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert SQLC models to store models
	result := make([]CronJob, len(sqlcCronJobs))
	for i, sqlcCron := range sqlcCronJobs {
		result[i] = CronJob{
			ID:           sqlcCron.ID,
			Name:         sqlcCron.Name,
			CronSpec:     sqlcCron.CronSpec,
			Namespace:    sqlcCron.Namespace,
			WorkflowName: sqlcCron.WorkflowName.String,
			Enabled:      sqlcCron.Enabled,
			CreatedAt:    sqlcCron.CreatedAt,
			UpdatedAt:    sqlcCron.UpdatedAt,
			LastRunAt:    nullInt64ToPtr(sqlcCron.LastRunAt),
			NextRunAt:    sqlcCron.NextRunAt,
		}
	}
	return result, nil
}

func (s *sqlcStore) UpdateCronJobLastRun(ctx context.Context, namespace, cronJobID string, lastRunAt, nextRunAt int64) error {
	now := s.clock.Now().UnixMilli()

	params := sqlcstore.UpdateCronJobLastRunParams{
		LastRunAt: sql.NullInt64{Int64: lastRunAt, Valid: true},
		NextRunAt: nextRunAt,
		UpdatedAt: now,
		ID:        cronJobID,
		Namespace: namespace,
	}

	return s.queries.UpdateCronJobLastRun(ctx, params)
}

func (s *sqlcStore) AcquireLease(ctx context.Context, lease *Lease) error {
	params := sqlcstore.CreateLeaseParams{
		ResourceID:  lease.ResourceID,
		Kind:        sqlcstore.LeasesKind(lease.Kind),
		Namespace:   lease.Namespace,
		WorkerID:    lease.WorkerID,
		AcquiredAt:  lease.AcquiredAt,
		ExpiresAt:   lease.ExpiresAt,
		HeartbeatAt: lease.HeartbeatAt,
	}

	err := s.queries.CreateLease(ctx, params)
	if err != nil {
		if isDuplicateKeyError(err) {
			return errors.New("lease already held by another worker")
		}
		return err
	}
	return nil
}

func (s *sqlcStore) HeartbeatLease(ctx context.Context, resourceID, workerID string, expiresAt int64) error {
	now := s.clock.Now().UnixMilli()

	params := sqlcstore.HeartbeatLeaseParams{
		HeartbeatAt: now,
		ExpiresAt:   expiresAt,
		ResourceID:  resourceID,
		WorkerID:    workerID,
	}

	// Execute the update - SQLC :exec queries don't return results for row counting
	// We'll trust that if no error occurs, the update was successful
	return s.queries.HeartbeatLease(ctx, params)
}

func (s *sqlcStore) ReleaseLease(ctx context.Context, resourceID, workerID string) error {
	params := sqlcstore.ReleaseLeaseParams{
		ResourceID: resourceID,
		WorkerID:   workerID,
	}

	return s.queries.ReleaseLease(ctx, params)
}

func (s *sqlcStore) GetLease(ctx context.Context, resourceID string) (*Lease, error) {
	params := sqlcstore.GetLeaseParams{
		ResourceID: resourceID,
		Kind:       "workflow", // Default to workflow kind for this use case
	}

	sqlcLease, err := s.queries.GetLease(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("lease not found")
		}
		return nil, err
	}

	// Convert SQLC model to store model
	result := &Lease{
		ResourceID:  sqlcLease.ResourceID,
		Kind:        string(sqlcLease.Kind),
		Namespace:   sqlcLease.Namespace,
		WorkerID:    sqlcLease.WorkerID,
		AcquiredAt:  sqlcLease.AcquiredAt,
		ExpiresAt:   sqlcLease.ExpiresAt,
		HeartbeatAt: sqlcLease.HeartbeatAt,
	}

	return result, nil
}

func (s *sqlcStore) CleanupExpiredLeases(ctx context.Context, namespace string) error {
	now := s.clock.Now().UnixMilli()

	params := sqlcstore.CleanupExpiredLeasesParams{
		Namespace: namespace,
		ExpiresAt: now,
	}

	return s.queries.CleanupExpiredLeases(ctx, params)
}

func (s *sqlcStore) GetExpiredLeases(ctx context.Context, namespace string) ([]Lease, error) {
	now := s.clock.Now().UnixMilli()

	params := sqlcstore.GetExpiredLeasesParams{
		Namespace: namespace,
		ExpiresAt: now,
	}

	sqlcLeases, err := s.queries.GetExpiredLeases(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert SQLC models to store models
	result := make([]Lease, len(sqlcLeases))
	for i, sqlcLease := range sqlcLeases {
		result[i] = Lease{
			ResourceID:  sqlcLease.ResourceID,
			Kind:        string(sqlcLease.Kind),
			Namespace:   sqlcLease.Namespace,
			WorkerID:    sqlcLease.WorkerID,
			AcquiredAt:  sqlcLease.AcquiredAt,
			ExpiresAt:   sqlcLease.ExpiresAt,
			HeartbeatAt: sqlcLease.HeartbeatAt,
		}
	}
	return result, nil
}

func (s *sqlcStore) ResetOrphanedWorkflows(ctx context.Context, namespace string) error {
	params := sqlcstore.ResetOrphanedWorkflowsParams{
		Namespace:   namespace,
		Namespace_2: namespace, // Same namespace used for both conditions
	}

	return s.queries.ResetOrphanedWorkflows(ctx, params)
}

func (s *sqlcStore) WithTx(ctx context.Context, fn func(Store) error) error {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create a new SQLC store instance with the transaction
	txQueries := s.queries.WithTx(tx)
	txStore := &sqlcStore{
		db:      s.db, // Keep the original DB for potential nested operations
		queries: txQueries,
		clock:   s.clock,
	}

	// Execute the function with the transactional store
	if err := fn(txStore); err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}

func (s *sqlcStore) GetAllWorkflows(ctx context.Context, namespace string) ([]WorkflowExecution, error) {
	sqlcWorkflows, err := s.queries.GetAllWorkflows(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// Convert SQLC models to store models (reusing conversion logic from GetPendingWorkflows)
	result := make([]WorkflowExecution, len(sqlcWorkflows))
	for i, sqlcWf := range sqlcWorkflows {
		result[i] = WorkflowExecution{
			ID:                sqlcWf.ID,
			WorkflowName:      sqlcWf.WorkflowName,
			Status:            WorkflowStatus(sqlcWf.Status),
			InputData:         sqlcWf.InputData,
			OutputData:        sqlcWf.OutputData,
			ErrorMessage:      sqlcWf.ErrorMessage.String,
			CreatedAt:         sqlcWf.CreatedAt,
			StartedAt:         nullInt64ToPtr(sqlcWf.StartedAt),
			CompletedAt:       nullInt64ToPtr(sqlcWf.CompletedAt),
			MaxAttempts:       sqlcWf.MaxAttempts,
			RemainingAttempts: sqlcWf.RemainingAttempts,
			NextRetryAt:       nullInt64ToPtr(sqlcWf.NextRetryAt),
			Namespace:         sqlcWf.Namespace,
			TriggerType:       TriggerType(sqlcWf.TriggerType.WorkflowExecutionsTriggerType),
			TriggerSource:     nullStringToPtr(sqlcWf.TriggerSource),
			SleepUntil:        nullInt64ToPtr(sqlcWf.SleepUntil),
			TraceID:           sqlcWf.TraceID.String,
			SpanID:            sqlcWf.SpanID.String,
		}
	}
	return result, nil
}

func (s *sqlcStore) GetAllSteps(ctx context.Context, namespace string) ([]WorkflowStep, error) {
	sqlcSteps, err := s.queries.GetAllSteps(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// Convert SQLC models to store models (reusing conversion logic from GetStep)
	result := make([]WorkflowStep, len(sqlcSteps))
	for i, sqlcStep := range sqlcSteps {
		result[i] = WorkflowStep{
			ID:                sqlcStep.ID,
			ExecutionID:       sqlcStep.ExecutionID,
			StepName:          sqlcStep.StepName,
			StepOrder:         sqlcStep.StepOrder,
			Status:            StepStatus(sqlcStep.Status),
			OutputData:        sqlcStep.OutputData,
			ErrorMessage:      sqlcStep.ErrorMessage.String,
			StartedAt:         nullInt64ToPtr(sqlcStep.StartedAt),
			CompletedAt:       nullInt64ToPtr(sqlcStep.CompletedAt),
			MaxAttempts:       sqlcStep.MaxAttempts,
			RemainingAttempts: sqlcStep.RemainingAttempts,
			Namespace:         sqlcStep.Namespace,
		}
	}
	return result, nil
}

// Helper functions for converting from nullable types
func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

func nullStringToPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}

func isDuplicateKeyError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate") ||
		strings.Contains(errStr, "UNIQUE constraint") ||
		strings.Contains(errStr, "PRIMARY KEY constraint")
}
