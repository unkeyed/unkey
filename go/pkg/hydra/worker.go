package hydra

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
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

	workflows, err := w.queryCircuitBreaker.Do(ctx, func(ctx context.Context) ([]store.WorkflowExecution, error) {
		return w.engine.store.GetPendingWorkflows(ctx, w.engine.namespace, fetchLimit, workflowNames)
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
			err := w.engine.store.AcquireWorkflowLease(ctx, workflow.ID, w.engine.namespace, w.config.WorkerID, w.config.ClaimTimeout)
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
			if err := w.engine.store.ReleaseLease(ctx, workflow.ID, w.config.WorkerID); err != nil {
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

func (w *worker) executeWorkflow(ctx context.Context, e *WorkflowExecution) {
	startTime := w.clock.Now()

	// Start tracing span for workflow execution
	var span trace.Span

	if e.TraceID != "" && e.SpanID != "" {
		// Reconstruct the exact trace context from stored trace ID and span ID
		traceID, traceErr := trace.TraceIDFromHex(e.TraceID)
		spanID, spanErr := trace.SpanIDFromHex(e.SpanID)

		if traceErr == nil && spanErr == nil {
			// Create the exact span context from the original workflow creation
			originalSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
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

	if e.TraceID != "" {
		spanAttributes = append(spanAttributes, attribute.String("hydra.original_trace_id", e.TraceID))
	}
	if e.SpanID != "" {
		spanAttributes = append(spanAttributes, attribute.String("hydra.original_span_id", e.SpanID))
	}

	span.SetAttributes(spanAttributes...)

	// Calculate queue time (time from creation to execution start)
	queueTime := time.Duration(startTime.UnixMilli()-e.CreatedAt) * time.Millisecond
	metrics.WorkflowQueueTimeSeconds.WithLabelValues(e.Namespace, e.WorkflowName).Observe(queueTime.Seconds())

	err := w.engine.store.UpdateWorkflowStatus(ctx, e.Namespace, e.ID, WorkflowStatusRunning, "")
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

		if failErr := w.engine.store.FailWorkflow(ctx, e.Namespace, e.ID, noHandlerErr.Error(), true); failErr != nil {
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
		ctx:             ctx, // This is the traced context from the worker span
		executionID:     e.ID,
		workflowName:    e.WorkflowName,
		namespace:       e.Namespace,
		workerID:        w.config.WorkerID,
		store:           w.engine.store,
		marshaller:      w.engine.marshaller,
		stepTimeout:     5 * time.Minute, // Default step timeout
		stepMaxAttempts: 3,               // Default step max attempts
		stepOrder:       0,
	}

	err = wf.Run(wctx, payload)

	if err != nil {
		tracing.RecordError(span, err)

		if suspendErr, ok := err.(*WorkflowSuspendedError); ok {
			span.SetAttributes(attribute.String("hydra.workflow.status", "suspended"))

			if sleepErr := w.engine.store.SleepWorkflow(ctx, e.Namespace, e.ID, suspendErr.ResumeTime); sleepErr != nil {
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

		if failErr := w.engine.store.FailWorkflow(ctx, e.Namespace, e.ID, err.Error(), isFinal); failErr != nil {
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

	span.SetAttributes(attribute.String("hydra.workflow.status", "completed"))

	if err := w.engine.store.CompleteWorkflow(ctx, e.Namespace, e.ID, nil); err != nil { // No output data for now
		tracing.RecordError(span, err)
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
			return nil, w.engine.store.HeartbeatLease(ctx, workflowID, w.config.WorkerID, newExpiresAt)
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
			err := w.engine.store.CleanupExpiredLeases(ctx, w.engine.namespace)
			if err != nil {
				w.engine.logger.Warn("Failed to cleanup expired leases", "error", err.Error())
			}

			// Then reset orphaned workflows back to pending so they can be picked up again
			err = w.engine.store.ResetOrphanedWorkflows(ctx, w.engine.namespace)
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

	dueCrons, err := w.engine.store.GetDueCronJobs(ctx, w.engine.namespace, now)
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

		lease := &Lease{
			ResourceID:  cronJob.ID,
			Kind:        string(LeaseKindCronJob),
			Namespace:   w.engine.namespace,
			WorkerID:    w.config.WorkerID,
			AcquiredAt:  now,
			ExpiresAt:   now + (5 * time.Minute).Milliseconds(), // 5 minute lease for cron execution
			HeartbeatAt: now,
		}

		err := w.engine.store.AcquireLease(ctx, lease)
		if err != nil {
			continue
		}

		w.executeCronJob(ctx, cronJob)

		if err := w.engine.store.ReleaseLease(ctx, cronJob.ID, w.config.WorkerID); err != nil {
			w.engine.logger.Error("Failed to release cron job lease",
				"cron_job_id", cronJob.ID,
				"cron_name", cronJob.Name,
				"worker_id", w.config.WorkerID,
				"error", err.Error(),
			)
		}
	}
}

func (w *worker) executeCronJob(ctx context.Context, cronJob CronJob) {

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
	if err := w.engine.store.UpdateCronJobLastRun(ctx, w.engine.namespace, cronJob.ID, now, nextRun); err != nil {
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
