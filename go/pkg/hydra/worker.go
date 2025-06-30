package hydra

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Worker represents a workflow worker that can start, run, and shutdown
type Worker interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// WorkerConfig holds the configuration for a worker
type WorkerConfig struct {
	WorkerID string

	Concurrency int

	PollInterval time.Duration

	HeartbeatInterval time.Duration

	ClaimTimeout time.Duration

	CronInterval time.Duration
}

type worker struct {
	engine    *Engine
	config    WorkerConfig
	workflows map[string]Workflow[any]
	shutdownC chan struct{}
	doneC     chan struct{}
	wg        sync.WaitGroup
}

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

	worker := &worker{
		engine:    e,
		config:    config,
		workflows: make(map[string]Workflow[any]),
		shutdownC: make(chan struct{}),
		doneC:     make(chan struct{}),
	}

	return worker, nil
}

func (w *worker) run(ctx context.Context) {
	defer close(w.doneC)

	w.wg.Add(4)
	go w.pollForWorkflows(ctx)
	go w.sendHeartbeats(ctx)
	go w.cleanupExpiredLeases(ctx)
	go w.processCronJobs(ctx)

	select {
	case <-w.shutdownC:
	case <-ctx.Done():
	}

	w.wg.Wait()
}

func (w *worker) pollForWorkflows(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	activeWorkflows := make(map[string]context.CancelFunc)
	var mu sync.Mutex

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			activeCount := len(activeWorkflows)
			mu.Unlock()

			if activeCount >= w.config.Concurrency {
				continue
			}

			w.pollOnce(ctx, &mu, activeWorkflows)

		case <-w.shutdownC:
			mu.Lock()
			for executionID, cancel := range activeWorkflows {
				cancel()
				delete(activeWorkflows, executionID)
			}
			mu.Unlock()
			return

		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) pollOnce(ctx context.Context, mu *sync.Mutex, activeWorkflows map[string]context.CancelFunc) {

	workflowNames := make([]string, 0, len(w.workflows))
	for name := range w.workflows {
		workflowNames = append(workflowNames, name)
	}

	workflows, err := w.engine.store.GetPendingWorkflows(ctx, w.engine.namespace, w.config.Concurrency, workflowNames)
	if err != nil {
		return
	}

	if len(workflows) == 0 {
		return
	}

	for _, workflow := range workflows {

		mu.Lock()
		if len(activeWorkflows) >= w.config.Concurrency {
			mu.Unlock()
			break
		}

		err := w.engine.store.AcquireWorkflowLease(ctx, workflow.ID, w.engine.namespace, w.config.WorkerID, w.config.ClaimTimeout)
		if err != nil {
			mu.Unlock()
			continue
		}

		workflowCtx, cancel := context.WithCancel(ctx)
		activeWorkflows[workflow.ID] = cancel
		mu.Unlock()

		w.wg.Add(1)
		go func(wf WorkflowExecution) {
			defer w.wg.Done()

			defer func() {
				mu.Lock()
				delete(activeWorkflows, wf.ID)
				mu.Unlock()
			}()

			w.executeWorkflow(workflowCtx, &wf)
		}(workflow)

	}
}

func (w *worker) executeWorkflow(ctx context.Context, e *WorkflowExecution) {

	err := w.engine.store.UpdateWorkflowStatus(ctx, e.Namespace, e.ID, WorkflowStatusRunning, "")
	if err != nil {
		return
	}

	wf, exists := w.workflows[e.WorkflowName]
	if !exists {
		err := fmt.Errorf("no handler registered for workflow %s", e.WorkflowName)

		w.engine.store.FailWorkflow(ctx, e.Namespace, e.ID, err.Error(), true)
		return
	}

	payload := &RawPayload{Data: e.InputData}

	wctx := &workflowContext{
		ctx:             ctx,
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
		if suspendErr, ok := err.(*WorkflowSuspendedError); ok {
			w.engine.store.SleepWorkflow(ctx, e.Namespace, e.ID, suspendErr.ResumeTime)
			return
		}

		isFinal := e.RemainingAttempts <= 1
		w.engine.store.FailWorkflow(ctx, e.Namespace, e.ID, err.Error(), isFinal)
		return
	}

	w.engine.store.CompleteWorkflow(ctx, e.Namespace, e.ID, nil) // No output data for now
}

func (w *worker) sendHeartbeats(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

		case <-w.shutdownC:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) cleanupExpiredLeases(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.HeartbeatInterval * 2) // Clean up less frequently than heartbeats
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := w.engine.store.CleanupExpiredLeases(ctx, w.engine.namespace)
			if err != nil {
				// Log error if needed
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

	ticker := time.NewTicker(w.config.CronInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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

		w.engine.store.ReleaseLease(ctx, cronJob.ID, w.config.WorkerID)
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

	handler(ctx, *payload)

	nextRun := calculateNextRun(cronJob.CronSpec, w.engine.clock.Now())
	w.engine.store.UpdateCronJobLastRun(ctx, w.engine.namespace, cronJob.ID, now, nextRun)

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
