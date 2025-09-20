package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra/metrics"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	activeLeases        map[string]bool                                          // Track workflow IDs we have leases for
	activeLeasesM       sync.RWMutex                                             // Protect the activeLeases map
	queryCircuitBreaker circuitbreaker.CircuitBreaker[[]store.WorkflowExecution] // Protect query operations
	leaseCircuitBreaker circuitbreaker.CircuitBreaker[any]                       // Protect lease operations
	workflowQueue       chan store.WorkflowExecution                             // Queue of workflows to process
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
	queryCircuitBreaker := circuitbreaker.New[[]store.WorkflowExecution]("hydra-query")
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
		workflowQueue:       make(chan store.WorkflowExecution, queueSize),
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
	if fetchLimit > 1000 {
		fetchLimit = 1000 // Maximum reasonable fetch size
	}

	// Convert to int32 safely for gosec - using string conversion to avoid overflow warning
	fetchLimit32, _ := strconv.ParseInt(strconv.Itoa(fetchLimit), 10, 32)

	workflows, err := w.queryCircuitBreaker.Do(ctx, func(ctx context.Context) ([]store.WorkflowExecution, error) {
		// Use new Query pattern
		now := time.Now().UnixMilli()
		var workflows []store.WorkflowExecution
		var err error

		if len(workflowNames) > 0 {
			// Use filtered query - for now just use the first workflow name
			// Multiple workflow names support requires SQLC query enhancement
			workflows, err = store.Query.GetPendingWorkflowsFiltered(ctx, w.engine.GetDB(), store.GetPendingWorkflowsFilteredParams{
				Namespace:    w.engine.namespace,
				NextRetryAt:  sql.NullInt64{Int64: now, Valid: true},
				SleepUntil:   sql.NullInt64{Int64: now, Valid: true},
				WorkflowName: workflowNames[0],
				Limit:        int32(fetchLimit32), //nolint:gosec // G115: fetchLimit is bounded to [10, 1000]
			})
		} else {
			workflows, err = store.Query.GetPendingWorkflows(ctx, w.engine.GetDB(), store.GetPendingWorkflowsParams{
				Namespace:   w.engine.namespace,
				NextRetryAt: sql.NullInt64{Int64: now, Valid: true},
				SleepUntil:  sql.NullInt64{Int64: now, Valid: true},
				Limit:       int32(fetchLimit32), //nolint:gosec // G115: fetchLimit is bounded to [10, 1000]
			})
		}

		if err != nil {
			return nil, err
		}

		// Return store types directly (no conversion needed)
		return workflows, nil
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
			// Try to acquire lease using new Query pattern with transaction
			err := w.acquireWorkflowLease(ctx, workflow.ID, w.config.WorkerID)
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
			// Use new Query pattern
			if err := store.Query.ReleaseLease(ctx, w.engine.GetDB(), store.ReleaseLeaseParams{
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

func (w *worker) executeWorkflow(ctx context.Context, e *store.WorkflowExecution) {
	startTime := w.clock.Now()

	// Start tracing span for workflow execution
	var span trace.Span

	if e.TraceID.Valid && e.SpanID.Valid && e.TraceID.String != "" && e.SpanID.String != "" {
		// Reconstruct the exact trace context from stored trace ID and span ID
		traceID, traceErr := trace.TraceIDFromHex(e.TraceID.String)
		spanID, spanErr := trace.SpanIDFromHex(e.SpanID.String)

		if traceErr == nil && spanErr == nil {
			// Create the exact span context from the original workflow creation
			originalSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
				TraceState: trace.TraceState{},
				Remote:     false,
			})

			// Set this context as the parent for the execution span
			ctx = trace.ContextWithSpanContext(ctx, originalSpanCtx)
		}
	}

	ctx, span = tracing.Start(ctx, fmt.Sprintf("hydra.worker.executeWorkflow.%s", e.WorkflowName))
	defer span.End()

	spanAttributes := []attribute.KeyValue{
		attribute.String("hydra.workflow.name", e.WorkflowName),
		attribute.String("hydra.execution.id", e.ID),
		attribute.String("hydra.namespace", e.Namespace),
		attribute.String("hydra.worker.id", w.config.WorkerID),
	}

	if e.TraceID.Valid && e.TraceID.String != "" {
		spanAttributes = append(spanAttributes, attribute.String("hydra.original_trace_id", e.TraceID.String))
	}
	if e.SpanID.Valid && e.SpanID.String != "" {
		spanAttributes = append(spanAttributes, attribute.String("hydra.original_span_id", e.SpanID.String))
	}

	span.SetAttributes(spanAttributes...)

	// Calculate queue time (time from creation to execution start)
	queueTime := time.Duration(startTime.UnixMilli()-e.CreatedAt) * time.Millisecond
	metrics.WorkflowQueueTimeSeconds.WithLabelValues(e.Namespace, e.WorkflowName).Observe(queueTime.Seconds())

	// Update workflow to running status with lease validation
	now := time.Now().UnixMilli()
	err := store.Query.UpdateWorkflowToRunning(ctx, w.engine.GetDB(), store.UpdateWorkflowToRunningParams{
		StartedAt:  sql.NullInt64{Int64: startTime.UnixMilli(), Valid: true},
		ID:         e.ID,
		Namespace:  e.Namespace,
		ResourceID: e.ID,
		WorkerID:   w.config.WorkerID,
		ExpiresAt:  now,
	})
	if err != nil {
		metrics.RecordError(e.Namespace, "worker", "status_update_failed")
		tracing.RecordError(span, err)
		span.SetAttributes(attribute.String("hydra.workflow.status", "failed"))
		return
	}

	wf, exists := w.workflows[e.WorkflowName]
	if !exists {
		noHandlerErr := fmt.Errorf("no handler registered for workflow %s", e.WorkflowName)
		tracing.RecordError(span, noHandlerErr)
		span.SetAttributes(attribute.String("hydra.workflow.status", "failed"))

		// Use lease-validated failure to ensure correctness
		failureTime := w.clock.Now().UnixMilli()
		result, failErr := w.engine.GetDB().ExecContext(ctx, `
			UPDATE workflow_executions
			SET status = 'failed', error_message = ?, remaining_attempts = remaining_attempts - 1, completed_at = ?, next_retry_at = NULL
			WHERE id = ? AND workflow_executions.namespace = ?
			  AND EXISTS (
			    SELECT 1 FROM leases
			    WHERE resource_id = ? AND kind = 'workflow'
			    AND worker_id = ? AND expires_at > ?
			  )`,
			sql.NullString{String: noHandlerErr.Error(), Valid: true},
			sql.NullInt64{Int64: failureTime, Valid: true},
			e.ID,
			e.Namespace,
			e.ID,              // resource_id for lease check
			w.config.WorkerID, // worker_id for lease check
			failureTime,       // expires_at check
		)
		if failErr != nil {
			w.engine.logger.Error("Failed to mark workflow as failed",
				"workflow_id", e.ID,
				"workflow_name", e.WorkflowName,
				"namespace", e.Namespace,
				"error", failErr.Error(),
			)
		} else {
			// Check if the failure actually happened (lease validation)
			if rowsAffected, checkErr := result.RowsAffected(); checkErr != nil {
				w.engine.logger.Error("Failed to check workflow failure result",
					"workflow_id", e.ID,
					"error", checkErr.Error(),
				)
			} else if rowsAffected == 0 {
				w.engine.logger.Warn("Workflow failure failed: lease expired or invalid",
					"workflow_id", e.ID,
					"worker_id", w.config.WorkerID,
				)
			}
		}
		metrics.ObserveWorkflowDuration(e.Namespace, e.WorkflowName, "failed", startTime)
		metrics.WorkflowsCompletedTotal.WithLabelValues(e.Namespace, e.WorkflowName, "failed").Inc()
		metrics.RecordError(e.Namespace, "worker", "no_handler_registered")
		return
	}

	payload := &RawPayload{Data: e.InputData}

	wctx := &workflowContext{
		ctx:             ctx, // This is the traced context from the worker span
		executionID:     e.ID,
		workflowName:    e.WorkflowName,
		namespace:       e.Namespace,
		workerID:        w.config.WorkerID,
		db:              w.engine.GetDB(),
		marshaller:      w.engine.marshaller,
		logger:          w.engine.logger.With("execution_id", e.ID, "namespace", e.Namespace, "workflow_name", e.WorkflowName),
		stepTimeout:     5 * time.Minute, // Default step timeout
		stepMaxAttempts: 3,               // Default step max attempts
	}

	err = wf.Run(wctx, payload)

	if err != nil {
		tracing.RecordError(span, err)

		if suspendErr, ok := err.(*WorkflowSuspendedError); ok {
			span.SetAttributes(attribute.String("hydra.workflow.status", "suspended"))

			// Use simple sleep workflow since we have the lease
			if sleepErr := store.Query.SleepWorkflow(ctx, w.engine.GetDB(), store.SleepWorkflowParams{
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
		span.SetAttributes(attribute.String("hydra.workflow.status", "failed"))

		// Use lease-validated failure to ensure correctness
		finalFailureTime := w.clock.Now().UnixMilli()
		var result sql.Result
		var failErr error

		if isFinal {
			// Final failure - no more retries
			result, failErr = w.engine.GetDB().ExecContext(ctx, `
				UPDATE workflow_executions
				SET status = 'failed', error_message = ?, remaining_attempts = remaining_attempts - 1, completed_at = ?, next_retry_at = NULL
				WHERE id = ? AND workflow_executions.namespace = ?
				  AND EXISTS (
				    SELECT 1 FROM leases
				    WHERE resource_id = ? AND kind = 'workflow'
				    AND worker_id = ? AND expires_at > ?
				  )`,
				sql.NullString{String: err.Error(), Valid: true},
				sql.NullInt64{Int64: finalFailureTime, Valid: true},
				e.ID,
				e.Namespace,
				e.ID,              // resource_id for lease check
				w.config.WorkerID, // worker_id for lease check
				finalFailureTime,  // expires_at check
			)
		} else {
			// Failure with retry - calculate next retry time
			nextRetryAt := w.clock.Now().Add(time.Duration(e.MaxAttempts-e.RemainingAttempts+1) * time.Second).UnixMilli()
			result, failErr = w.engine.GetDB().ExecContext(ctx, `
				UPDATE workflow_executions
				SET status = 'failed', error_message = ?, remaining_attempts = remaining_attempts - 1, next_retry_at = ?
				WHERE id = ? AND workflow_executions.namespace = ?
				  AND EXISTS (
				    SELECT 1 FROM leases
				    WHERE resource_id = ? AND kind = 'workflow'
				    AND worker_id = ? AND expires_at > ?
				  )`,
				sql.NullString{String: err.Error(), Valid: true},
				sql.NullInt64{Int64: nextRetryAt, Valid: true},
				e.ID,
				e.Namespace,
				e.ID,              // resource_id for lease check
				w.config.WorkerID, // worker_id for lease check
				finalFailureTime,  // expires_at check
			)
		}
		if failErr != nil {
			w.engine.logger.Error("Failed to mark workflow as failed",
				"workflow_id", e.ID,
				"workflow_name", e.WorkflowName,
				"namespace", e.Namespace,
				"is_final", isFinal,
				"original_error", err.Error(),
				"fail_error", failErr.Error(),
			)
		} else {
			// Check if the failure actually happened (lease validation)
			if rowsAffected, checkErr := result.RowsAffected(); checkErr != nil {
				w.engine.logger.Error("Failed to check workflow failure result",
					"workflow_id", e.ID,
					"error", checkErr.Error(),
				)
			} else if rowsAffected == 0 {
				w.engine.logger.Warn("Workflow failure failed: lease expired or invalid",
					"workflow_id", e.ID,
					"worker_id", w.config.WorkerID,
					"is_final", isFinal,
				)
			}
		}

		if !isFinal {
			metrics.WorkflowsRetriedTotal.WithLabelValues(e.Namespace, e.WorkflowName, fmt.Sprintf("%d", e.MaxAttempts-e.RemainingAttempts+1)).Inc()
		}

		metrics.ObserveWorkflowDuration(e.Namespace, e.WorkflowName, "failed", startTime)
		metrics.WorkflowsCompletedTotal.WithLabelValues(e.Namespace, e.WorkflowName, "failed").Inc()
		return
	}

	span.SetAttributes(attribute.String("hydra.workflow.status", "completed"))

	// Use lease-validated completion to ensure correctness
	now = w.clock.Now().UnixMilli()
	result, err := w.engine.GetDB().ExecContext(ctx, `
		UPDATE workflow_executions
		SET status = 'completed', completed_at = ?, output_data = ?
		WHERE id = ? AND workflow_executions.namespace = ?
		  AND EXISTS (
		    SELECT 1 FROM leases
		    WHERE resource_id = ? AND kind = 'workflow'
		    AND worker_id = ? AND expires_at > ?
		  )`,
		sql.NullInt64{Int64: now, Valid: true},
		[]byte{}, // No output data for now
		e.ID,
		e.Namespace,
		e.ID,              // resource_id for lease check
		w.config.WorkerID, // worker_id for lease check
		now,               // expires_at check
	)
	if err != nil {
		tracing.RecordError(span, err)
		w.engine.logger.Error("Failed to mark workflow as completed",
			"workflow_id", e.ID,
			"workflow_name", e.WorkflowName,
			"namespace", e.Namespace,
			"error", err.Error(),
		)
		return
	}

	// Check if the completion actually happened (lease validation)
	rowsAffected, checkErr := result.RowsAffected()
	if checkErr != nil {
		w.engine.logger.Error("Failed to check workflow completion result",
			"workflow_id", e.ID,
			"error", checkErr.Error(),
		)
		return
	}
	if rowsAffected == 0 {
		w.engine.logger.Warn("Workflow completion failed: lease expired or invalid",
			"workflow_id", e.ID,
			"worker_id", w.config.WorkerID,
		)
		return
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
			// Use new Query pattern
			return nil, store.Query.HeartbeatLease(ctx, w.engine.GetDB(), store.HeartbeatLeaseParams{
				HeartbeatAt: now,
				ExpiresAt:   newExpiresAt,
				ResourceID:  workflowID,
				WorkerID:    w.config.WorkerID,
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
			now := w.clock.Now().UnixMilli()
			err := store.Query.CleanupExpiredLeases(ctx, w.engine.GetDB(), store.CleanupExpiredLeasesParams{
				Namespace: w.engine.namespace,
				ExpiresAt: now,
			})
			if err != nil {
				w.engine.logger.Warn("Failed to cleanup expired leases", "error", err.Error())
			}

			// Then reset orphaned workflows back to pending so they can be picked up again
			err = store.Query.ResetOrphanedWorkflows(ctx, w.engine.GetDB(), store.ResetOrphanedWorkflowsParams{
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

	dueCrons, err := store.Query.GetDueCronJobs(ctx, w.engine.GetDB(), store.GetDueCronJobsParams{
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
		if cronJob.WorkflowName.Valid && cronJob.WorkflowName.String != "" {
			_, canHandle = w.workflows[cronJob.WorkflowName.String]
		} else {
			_, canHandle = w.engine.cronHandlers[cronJob.Name]
		}

		if !canHandle {
			continue
		}

		err := store.Query.CreateLease(ctx, w.engine.GetDB(), store.CreateLeaseParams{
			ResourceID:  cronJob.ID,
			Kind:        store.LeasesKindCronJob,
			Namespace:   w.engine.namespace,
			WorkerID:    w.config.WorkerID,
			AcquiredAt:  now,
			ExpiresAt:   now + (5 * time.Minute).Milliseconds(), // 5 minute lease for cron execution
			HeartbeatAt: now,
		})
		if err != nil {
			continue
		}

		w.executeCronJob(ctx, cronJob)

		if err := store.Query.ReleaseLease(ctx, w.engine.GetDB(), store.ReleaseLeaseParams{
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

func (w *worker) executeCronJob(ctx context.Context, cronJob store.CronJob) {

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

	// Update cron job with lease validation - only if worker holds valid cron lease
	nextRun := calculateNextRun(cronJob.CronSpec, w.engine.clock.Now())
	updateTime := w.engine.clock.Now().UnixMilli()
	result, err := w.engine.GetDB().ExecContext(ctx, `
		UPDATE cron_jobs
		SET last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ? AND namespace = ?
		  AND EXISTS (
		    SELECT 1 FROM leases
		    WHERE resource_id = ? AND kind = 'cron_job'
		    AND worker_id = ? AND expires_at > ?
		  )`,
		sql.NullInt64{Int64: now, Valid: true},
		nextRun,
		updateTime,
		cronJob.ID,
		w.engine.namespace,
		cronJob.ID,        // resource_id for lease check
		w.config.WorkerID, // worker_id for lease check
		updateTime,        // expires_at check
	)
	if err != nil {
		w.engine.logger.Error("Failed to update cron job last run time",
			"cron_job_id", cronJob.ID,
			"cron_name", cronJob.Name,
			"namespace", w.engine.namespace,
			"last_run", now,
			"next_run", nextRun,
			"error", err.Error(),
		)
	} else {
		// Check if the update actually happened (lease validation)
		if rowsAffected, checkErr := result.RowsAffected(); checkErr != nil {
			w.engine.logger.Error("Failed to check cron job update result",
				"cron_job_id", cronJob.ID,
				"error", checkErr.Error(),
			)
		} else if rowsAffected == 0 {
			w.engine.logger.Warn("Cron job update failed: lease expired or invalid",
				"cron_job_id", cronJob.ID,
				"worker_id", w.config.WorkerID,
			)
		}
	}

}

// acquireWorkflowLease implements workflow lease acquisition using new Query pattern
func (w *worker) acquireWorkflowLease(ctx context.Context, workflowID, workerID string) error {
	now := w.clock.Now().UnixMilli()
	expiresAt := now + w.config.ClaimTimeout.Milliseconds()

	// Begin transaction
	tx, err := w.engine.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			w.engine.logger.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	// First, check if workflow is still available for leasing
	workflow, err := store.Query.GetWorkflow(ctx, tx, store.GetWorkflowParams{
		ID:        workflowID,
		Namespace: w.engine.namespace,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fmt.Errorf("workflow not found")
		}
		return err
	}

	// Check if workflow is in a valid state for execution
	if workflow.Status != store.WorkflowExecutionsStatusPending &&
		workflow.Status != store.WorkflowExecutionsStatusFailed &&
		workflow.Status != store.WorkflowExecutionsStatusSleeping {
		return fmt.Errorf("workflow not available for execution, status: %s", workflow.Status)
	}

	// Check for retry timing if it's a failed workflow
	if workflow.Status == store.WorkflowExecutionsStatusFailed &&
		workflow.NextRetryAt.Valid && workflow.NextRetryAt.Int64 > now {
		return fmt.Errorf("workflow retry not yet due")
	}

	// Check for sleep timing if it's a sleeping workflow
	if workflow.Status == store.WorkflowExecutionsStatusSleeping &&
		workflow.SleepUntil.Valid && workflow.SleepUntil.Int64 > now {
		return fmt.Errorf("workflow still sleeping")
	}

	// Try to create the lease
	err = store.Query.CreateLease(ctx, tx, store.CreateLeaseParams{
		ResourceID:  workflowID,
		Kind:        store.LeasesKindWorkflow,
		Namespace:   w.engine.namespace,
		WorkerID:    workerID,
		AcquiredAt:  now,
		ExpiresAt:   expiresAt,
		HeartbeatAt: now,
	})
	if err != nil {
		// If lease creation failed, try to take over ONLY expired leases
		leaseResult, leaseErr := tx.ExecContext(ctx, `
			UPDATE leases
			SET worker_id = ?, acquired_at = ?, expires_at = ?, heartbeat_at = ?
			WHERE resource_id = ? AND kind = ? AND expires_at < ?`,
			workerID, now, expiresAt, now, workflowID, store.LeasesKindWorkflow, now)
		if leaseErr != nil {
			return fmt.Errorf("failed to check for expired lease: %w", leaseErr)
		}

		// Check if we actually took over an expired lease
		rowsAffected, rowsErr := leaseResult.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("failed to check lease takeover result: %w", rowsErr)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("workflow is already leased by another worker")
		}
	}

	// Update workflow to running status
	err = store.Query.UpdateWorkflowToRunning(ctx, tx, store.UpdateWorkflowToRunningParams{
		StartedAt:  sql.NullInt64{Int64: now, Valid: true},
		ID:         workflowID,
		Namespace:  w.engine.namespace,
		ResourceID: workflowID,
		WorkerID:   w.config.WorkerID,
		ExpiresAt:  now,
	})
	if err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	// Commit the transaction
	return tx.Commit()
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
