package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/db"
	"github.com/unkeyed/unkey/go/pkg/hydra/metrics"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Worker represents a workflow worker that can start, run, and shutdown.
//
// Workers are responsible for:
// - Polling the database for pending workflows
// - Acquiring exclusive leases on workflows to prevent duplicate execution
// - Executing workflow logic by calling registered workflow handlers
// - Sending periodic heartbeats to maintain lease ownership
// - Processing scheduled cron jobs
// - Recording metrics for observability
//
// Workers are designed to be run as long-lived processes and can safely
// handle network failures, database outages, and graceful shutdowns.
type Worker interface {
	// Start begins the worker's main execution loop.
	// This method blocks until the context is cancelled or an error occurs.
	Start(ctx context.Context) error

	// Shutdown gracefully stops the worker and waits for active workflows to complete.
	// This method should be called during application shutdown to ensure clean termination.
	Shutdown(ctx context.Context) error
}

// WorkerConfig holds the configuration for a worker instance.
//
// All fields are optional and will use sensible defaults if not specified.
type WorkerConfig struct {
	// WorkerID uniquely identifies this worker instance.
	// If not provided, a random ID will be generated.
	WorkerID string

	// Concurrency controls how many workflows can execute simultaneously.
	// Defaults to 10 if not specified.
	Concurrency int

	// PollInterval controls how frequently the worker checks for new workflows.
	// Shorter intervals provide lower latency but increase database load.
	// Defaults to 5 seconds if not specified.
	PollInterval time.Duration

	// HeartbeatInterval controls how frequently the worker sends lease heartbeats.
	// This should be significantly shorter than ClaimTimeout to prevent lease expiration.
	// Defaults to 30 seconds if not specified.
	HeartbeatInterval time.Duration

	// ClaimTimeout controls how long a worker can hold a workflow lease.
	// Expired leases are automatically released, allowing other workers to take over.
	// Defaults to 5 minutes if not specified.
	ClaimTimeout time.Duration

	// CronInterval controls how frequently the worker checks for due cron jobs.
	// Defaults to 1 minute if not specified.
	CronInterval time.Duration
}

type worker struct {
	engine              *Engine
	config              WorkerConfig
	workflows           map[string]Workflow[any]
	clock               clock.Clock
	shutdownC           chan struct{}
	doneC               chan struct{}
	wg                  sync.WaitGroup
	activeLeases        map[string]bool                                       // Track workflow IDs we have leases for
	activeLeasesM       sync.RWMutex                                          // Protect the activeLeases map
	queryCircuitBreaker circuitbreaker.CircuitBreaker[[]db.WorkflowExecution] // Protect query operations
	leaseCircuitBreaker circuitbreaker.CircuitBreaker[any]                    // Protect lease operations
	workflowQueue       chan db.WorkflowExecution                             // Queue of workflows to process
}

// NewWorker creates a new worker instance with the provided configuration.
//
// The worker will be associated with the given engine and inherit its
// namespace and storage configuration. Missing configuration values
// will be populated with sensible defaults.
//
// The worker must have workflows registered using RegisterWorkflow()
// before calling Start().
//
// Example:
//
//	worker, err := hydra.NewWorker(engine, hydra.WorkerConfig{
//	    WorkerID:          "worker-1",
//	    Concurrency:       20,
//	    PollInterval:      100 * time.Millisecond,
//	    HeartbeatInterval: 30 * time.Second,
//	    ClaimTimeout:      5 * time.Minute,
//	})
//	if err != nil {
//	    return err
//	}
//
// The worker includes built-in circuit breakers to protect against
// database overload and automatic retry logic for transient failures.
func NewWorker(e *Engine, config WorkerConfig) (Worker, error) {
	if config.WorkerID == "" {
		config.WorkerID = uid.New(uid.WorkerPrefix)
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.ClaimTimeout <= 0 {
		config.ClaimTimeout = 5 * time.Minute
	}
	if config.CronInterval <= 0 {
		config.CronInterval = 1 * time.Minute
	}

	// Initialize circuit breakers for different database operations
	queryCircuitBreaker := circuitbreaker.New[[]db.WorkflowExecution]("hydra-query")
	leaseCircuitBreaker := circuitbreaker.New[any]("hydra-lease")

	// Create workflow queue with capacity based on concurrency
	queueSize := config.Concurrency * 10
	if queueSize < 50 {
		queueSize = 50 // Minimum queue size
	}

	worker := &worker{
		engine:              e,
		config:              config,
		workflows:           make(map[string]Workflow[any]),
		clock:               e.clock,
		shutdownC:           make(chan struct{}),
		doneC:               make(chan struct{}),
		wg:                  sync.WaitGroup{},
		activeLeases:        make(map[string]bool),
		activeLeasesM:       sync.RWMutex{},
		queryCircuitBreaker: queryCircuitBreaker,
		leaseCircuitBreaker: leaseCircuitBreaker,
		workflowQueue:       make(chan db.WorkflowExecution, queueSize),
	}

	return worker, nil
}

func (w *worker) run(ctx context.Context) {
	defer close(w.doneC)

	// Start workflow processors
	for i := 0; i < w.config.Concurrency; i++ {
		w.wg.Add(1)
		go w.processWorkflows(ctx)
	}

	w.wg.Add(4)
	go w.pollForWorkflows(ctx)
	go w.sendHeartbeats(ctx)
	go w.cleanupExpiredLeases(ctx)
	go w.processCronJobs(ctx)

	select {
	case <-w.shutdownC:
	case <-ctx.Done():
	}

	// Don't close the queue immediately - let processors drain it first
	w.wg.Wait()
}

func (w *worker) pollForWorkflows(ctx context.Context) {
	defer w.wg.Done()

	ticker := w.clock.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	tickerC := ticker.C()

	for {
		select {
		case <-tickerC:
			w.pollOnce(ctx)

		case <-w.shutdownC:
			return

		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) pollOnce(ctx context.Context) {
	workflowNames := make([]string, 0, len(w.workflows))
	for name := range w.workflows {
		workflowNames = append(workflowNames, name)
	}

	// Use a more conservative fetch limit to reduce contention
	fetchLimit := w.config.Concurrency * 2 // Fetch less to reduce contention
	if fetchLimit < 10 {
		fetchLimit = 10 // Minimum fetch size
	}

	workflows, err := w.queryCircuitBreaker.Do(ctx, func(ctx context.Context) ([]db.WorkflowExecution, error) {
		now := w.clock.Now().UnixMilli()
		return db.Query.GetPendingWorkflows(ctx, w.engine.db, db.GetPendingWorkflowsParams{
			Namespace:   w.engine.namespace,
			NextRetryAt: sql.NullInt64{Int64: now, Valid: true},
			SleepUntil:  sql.NullInt64{Int64: now, Valid: true},
			Limit:       int32(fetchLimit),
			Offset:      0,
		})
	})

	// Record polling metrics
	if err != nil {
		metrics.WorkerPollsTotal.WithLabelValues(w.config.WorkerID, w.engine.namespace, "error").Inc()
		return
	}

	// Record successful poll with found work status
	status := "no_work"
	if len(workflows) > 0 {
		status = "found_work"
	}
	metrics.WorkerPollsTotal.WithLabelValues(w.config.WorkerID, w.engine.namespace, status).Inc()

	// Queue workflows - let polling goroutine block if needed
	for _, workflow := range workflows {
		w.workflowQueue <- workflow
	}
}

func (w *worker) processWorkflows(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case workflow := <-w.workflowQueue:
			// Try to acquire lease with direct store call (no circuit breaker to avoid blocking)
			expiresAt := w.clock.Now().Add(w.config.ClaimTimeout).UnixMilli()
			err := db.Query.AcquireLease(ctx, w.engine.db, db.AcquireLeaseParams{
				ResourceID:  workflow.ID,
				Kind:        db.LeasesKindWorkflow,
				Namespace:   w.engine.namespace,
				WorkerID:    w.config.WorkerID,
				AcquiredAt:  w.clock.Now().UnixMilli(),
				ExpiresAt:   expiresAt,
				HeartbeatAt: w.clock.Now().UnixMilli(),
			})
			if err != nil {
				// Another worker got it or error, skip this workflow
				metrics.LeaseAcquisitionsTotal.WithLabelValues(w.config.WorkerID, "workflow", "failed").Inc()
				continue
			}

			// Record successful lease acquisition
			metrics.LeaseAcquisitionsTotal.WithLabelValues(w.config.WorkerID, "workflow", "success").Inc()

			// Track this lease for heartbeats
			w.addActiveLease(workflow.ID)

			// Update active workflows gauge
			metrics.WorkflowsActive.WithLabelValues(w.engine.namespace, w.config.WorkerID).Inc()

			// Execute the workflow
			w.executeWorkflow(ctx, &workflow)

			// Release the lease and stop tracking it
			if _, err := db.Query.ReleaseLease(ctx, w.engine.db, db.ReleaseLeaseParams{
				ResourceID: workflow.ID,
				WorkerID:   w.config.WorkerID,
			}); err != nil {
				w.engine.logger.Error("Failed to release workflow lease",
					"workflow_id", workflow.ID,
					"worker_id", w.config.WorkerID,
					"error", err.Error(),
				)
			}
			w.removeActiveLease(workflow.ID)

			// Update active workflows gauge
			metrics.WorkflowsActive.WithLabelValues(w.engine.namespace, w.config.WorkerID).Dec()

		case <-w.shutdownC:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) executeWorkflow(ctx context.Context, e *db.WorkflowExecution) {
	startTime := w.clock.Now()

	// Calculate queue time (time from creation to execution start)
	queueTime := time.Duration(startTime.UnixMilli()-e.CreatedAt) * time.Millisecond
	metrics.WorkflowQueueTimeSeconds.WithLabelValues(e.Namespace, e.WorkflowName).Observe(queueTime.Seconds())

	err := db.Query.UpdateWorkflowStatusRunning(ctx, w.engine.db, db.UpdateWorkflowStatusRunningParams{
		StartedAt: sql.NullInt64{Int64: w.clock.Now().UnixMilli(), Valid: true},
		ID:        e.ID,
		Namespace: e.Namespace,
	})
	if err != nil {
		metrics.RecordError(e.Namespace, "worker", "status_update_failed")
		return
	}

	wf, exists := w.workflows[e.WorkflowName]
	if !exists {
		noHandlerErr := fmt.Errorf("no handler registered for workflow %s", e.WorkflowName)
		if failErr := db.Query.UpdateWorkflowStatus(ctx, w.engine.db, db.UpdateWorkflowStatusParams{
			Status:       db.WorkflowExecutionsStatusFailed,
			ErrorMessage: sql.NullString{String: noHandlerErr.Error(), Valid: true},
			ID:           e.ID,
			Namespace:    e.Namespace,
		}); failErr != nil {
			w.engine.logger.Error("Failed to mark workflow as failed",
				"workflow_id", e.ID,
				"workflow_name", e.WorkflowName,
				"namespace", e.Namespace,
				"error", failErr.Error(),
			)
		}
		metrics.ObserveWorkflowDuration(e.Namespace, e.WorkflowName, "failed", startTime)
		metrics.WorkflowsCompletedTotal.WithLabelValues(e.Namespace, e.WorkflowName, "failed").Inc()
		metrics.RecordError(e.Namespace, "worker", "no_handler_registered")
		return
	}

	payload := &RawPayload{Data: e.InputData}

	wctx := &workflowContext{
		ctx:             ctx,
		executionID:     e.ID,
		workflowName:    e.WorkflowName,
		namespace:       e.Namespace,
		workerID:        w.config.WorkerID,
		db:              w.engine.db,
		marshaller:      w.engine.marshaller,
		stepTimeout:     5 * time.Minute, // Default step timeout
		stepMaxAttempts: 3,               // Default step max attempts
		stepOrder:       0,
	}

	err = wf.Run(wctx, payload)

	if err != nil {
		if suspendErr, ok := err.(*WorkflowSuspendedError); ok {
			if sleepErr := db.Query.SleepWorkflow(ctx, w.engine.db, db.SleepWorkflowParams{
				SleepUntil: sql.NullInt64{Int64: suspendErr.ResumeTime, Valid: true},
				ID:         e.ID,
				Namespace:  e.Namespace,
			}); sleepErr != nil {
				w.engine.logger.Error("Failed to suspend workflow",
					"workflow_id", e.ID,
					"workflow_name", e.WorkflowName,
					"namespace", e.Namespace,
					"resume_time", suspendErr.ResumeTime,
					"error", sleepErr.Error(),
				)
			}
			metrics.SleepsStartedTotal.WithLabelValues(e.Namespace, e.WorkflowName).Inc()
			return
		}

		isFinal := e.RemainingAttempts <= 1
		if failErr := db.Query.UpdateWorkflowStatus(ctx, w.engine.db, db.UpdateWorkflowStatusParams{
			Status:       db.WorkflowExecutionsStatusFailed,
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
			ID:           e.ID,
			Namespace:    e.Namespace,
		}); failErr != nil {
			w.engine.logger.Error("Failed to mark workflow as failed",
				"workflow_id", e.ID,
				"workflow_name", e.WorkflowName,
				"namespace", e.Namespace,
				"is_final", isFinal,
				"original_error", err.Error(),
				"fail_error", failErr.Error(),
			)
		}

		if !isFinal {
			metrics.WorkflowsRetriedTotal.WithLabelValues(e.Namespace, e.WorkflowName, fmt.Sprintf("%d", e.MaxAttempts-e.RemainingAttempts+1)).Inc()
		}

		metrics.ObserveWorkflowDuration(e.Namespace, e.WorkflowName, "failed", startTime)
		metrics.WorkflowsCompletedTotal.WithLabelValues(e.Namespace, e.WorkflowName, "failed").Inc()
		return
	}

	if err := db.Query.CompleteWorkflow(ctx, w.engine.db, db.CompleteWorkflowParams{
		CompletedAt: sql.NullInt64{Int64: w.clock.Now().UnixMilli(), Valid: true},
		OutputData:  nil, // No output data for now
		ID:          e.ID,
		Namespace:   e.Namespace,
	}); err != nil {
		w.engine.logger.Error("Failed to mark workflow as completed",
			"workflow_id", e.ID,
			"workflow_name", e.WorkflowName,
			"namespace", e.Namespace,
			"error", err.Error(),
		)
	}
	metrics.ObserveWorkflowDuration(e.Namespace, e.WorkflowName, "completed", startTime)
	metrics.WorkflowsCompletedTotal.WithLabelValues(e.Namespace, e.WorkflowName, "completed").Inc()
}

func (w *worker) sendHeartbeats(ctx context.Context) {
	defer w.wg.Done()

	ticker := w.clock.NewTicker(w.config.HeartbeatInterval)
	defer ticker.Stop()
	tickerC := ticker.C()

	for {
		select {
		case <-tickerC:
			w.sendHeartbeatsForActiveLeases(ctx)

		case <-w.shutdownC:
			return
		case <-ctx.Done():
			return
		}
	}
}

// addActiveLease tracks a workflow lease for heartbeat sending
func (w *worker) addActiveLease(workflowID string) {
	w.activeLeasesM.Lock()
	defer w.activeLeasesM.Unlock()
	w.activeLeases[workflowID] = true
}

// removeActiveLease stops tracking a workflow lease
func (w *worker) removeActiveLease(workflowID string) {
	w.activeLeasesM.Lock()
	defer w.activeLeasesM.Unlock()
	delete(w.activeLeases, workflowID)
}

// sendHeartbeatsForActiveLeases sends heartbeats for all workflows this worker has leases for
func (w *worker) sendHeartbeatsForActiveLeases(ctx context.Context) {
	w.activeLeasesM.RLock()
	// Copy the map to avoid holding the lock while sending heartbeats
	leaseIDs := make([]string, 0, len(w.activeLeases))
	for workflowID := range w.activeLeases {
		leaseIDs = append(leaseIDs, workflowID)
	}
	w.activeLeasesM.RUnlock()

	// Send heartbeats for each active lease
	now := w.clock.Now().UnixMilli()
	newExpiresAt := now + w.config.ClaimTimeout.Milliseconds()

	for _, workflowID := range leaseIDs {
		// Protect heartbeat with circuit breaker
		_, err := w.leaseCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
			return nil, db.Query.UpdateLease(ctx, w.engine.db, db.UpdateLeaseParams{
				ExpiresAt:   newExpiresAt,
				HeartbeatAt: w.clock.Now().UnixMilli(),
				ResourceID:  workflowID,
			})
		})
		if err != nil {
			// Record failed heartbeat
			metrics.WorkerHeartbeatsTotal.WithLabelValues(w.config.WorkerID, w.engine.namespace, "failed").Inc()
			continue
		}

		// Record successful heartbeat
		metrics.WorkerHeartbeatsTotal.WithLabelValues(w.config.WorkerID, w.engine.namespace, "success").Inc()
	}
}

func (w *worker) cleanupExpiredLeases(ctx context.Context) {
	defer w.wg.Done()

	ticker := w.clock.NewTicker(w.config.HeartbeatInterval * 2) // Clean up less frequently than heartbeats
	defer ticker.Stop()
	tickerC := ticker.C()

	for {
		select {
		case <-tickerC:
			// Clean up expired leases first
			err := db.Query.CleanupExpiredLeases(ctx, w.engine.db, db.CleanupExpiredLeasesParams{
				Namespace: w.engine.namespace,
				ExpiresAt: w.clock.Now().UnixMilli(),
			})
			if err != nil {
				w.engine.logger.Warn("Failed to cleanup expired leases", "error", err.Error())
			}

			// Then reset orphaned workflows back to pending so they can be picked up again
			err = db.Query.ResetOrphanedWorkflows(ctx, w.engine.db, db.ResetOrphanedWorkflowsParams{
				Namespace:   w.engine.namespace,
				Namespace_2: w.engine.namespace,
			})
			if err != nil {
				w.engine.logger.Warn("Failed to reset orphaned workflows", "error", err.Error())
			}

		case <-w.shutdownC:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) processCronJobs(ctx context.Context) {
	defer w.wg.Done()

	ticker := w.clock.NewTicker(w.config.CronInterval)
	defer ticker.Stop()
	tickerC := ticker.C()

	for {
		select {
		case <-tickerC:
			w.processDueCronJobs(ctx)

		case <-w.shutdownC:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) processDueCronJobs(ctx context.Context) {

	now := w.engine.clock.Now().UnixMilli()

	dueCrons, err := db.Query.GetDueCronJobs(ctx, w.engine.db, db.GetDueCronJobsParams{
		Namespace: w.engine.namespace,
		NextRunAt: now,
	})
	if err != nil {
		return
	}

	if len(dueCrons) == 0 {
		return
	}

	for _, cronJob := range dueCrons {
		var canHandle bool
		if cronJob.WorkflowName != "" {
			_, canHandle = w.workflows[cronJob.WorkflowName]
		} else {
			_, canHandle = w.engine.cronHandlers[cronJob.Name]
		}

		if !canHandle {
			continue
		}

		err := db.Query.AcquireLease(ctx, w.engine.db, db.AcquireLeaseParams{
			ResourceID:  cronJob.ID,
			Kind:        db.LeasesKindCronJob,
			Namespace:   w.engine.namespace,
			WorkerID:    w.config.WorkerID,
			AcquiredAt:  w.clock.Now().UnixMilli(),
			ExpiresAt:   w.clock.Now().Add(w.config.ClaimTimeout).UnixMilli(),
			HeartbeatAt: w.clock.Now().UnixMilli(),
		})
		if err != nil {
			continue
		}

		w.executeCronJob(ctx, cronJob)

		if _, err := db.Query.ReleaseLease(ctx, w.engine.db, db.ReleaseLeaseParams{
			ResourceID: cronJob.ID,
			WorkerID:   w.config.WorkerID,
		}); err != nil {
			w.engine.logger.Error("Failed to release cron job lease",
				"cron_job_id", cronJob.ID,
				"cron_name", cronJob.Name,
				"worker_id", w.config.WorkerID,
				"error", err.Error(),
			)
		}
	}
}

func (w *worker) executeCronJob(ctx context.Context, cronJob db.CronJob) {

	now := w.engine.clock.Now().UnixMilli()

	payload := &CronPayload{
		CronJobID:   cronJob.ID,
		CronName:    cronJob.Name,
		ScheduledAt: cronJob.NextRunAt,
		ActualRunAt: now,
		Namespace:   cronJob.Namespace,
	}

	handler, exists := w.engine.cronHandlers[cronJob.Name]
	if !exists {
		return
	}

	// Execute cron handler with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				w.engine.logger.Error("Cron handler panicked",
					"cron_job_id", cronJob.ID,
					"cron_name", cronJob.Name,
					"panic", r,
				)
			}
		}()
		if err := handler(ctx, *payload); err != nil {
			w.engine.logger.Error("Cron handler execution failed",
				"cron_job_id", cronJob.ID,
				"cron_name", cronJob.Name,
				"error", err.Error(),
			)
		}
	}()

	nextRun := calculateNextRun(cronJob.CronSpec, w.engine.clock.Now())
	if err := db.Query.UpdateCronJob(ctx, w.engine.db, db.UpdateCronJobParams{
		CronSpec:     cronJob.CronSpec,
		WorkflowName: cronJob.WorkflowName,
		Enabled:      cronJob.Enabled,
		UpdatedAt:    now,
		NextRunAt:    nextRun,
		ID:           cronJob.ID,
	}); err != nil {
		w.engine.logger.Error("Failed to update cron job last run time",
			"cron_job_id", cronJob.ID,
			"cron_name", cronJob.Name,
			"namespace", w.engine.namespace,
			"last_run", now,
			"next_run", nextRun,
			"error", err.Error(),
		)
	}

}

func (w *worker) Start(ctx context.Context) error {
	go w.run(ctx)
	return nil
}

func (w *worker) Shutdown(ctx context.Context) error {
	select {
	case <-w.shutdownC:
	default:
		close(w.shutdownC)
	}

	select {
	case <-w.doneC:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
