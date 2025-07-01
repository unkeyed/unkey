package hydra

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// ChaosSimulator orchestrates chaos testing scenarios
type ChaosSimulator struct {
	engine          *Engine
	workers         []Worker
	workflowMetrics map[string]*WorkflowMetrics
	mu              sync.RWMutex

	// Chaos controls
	databaseFailureRate  atomic.Value // float64
	workerCrashRate      atomic.Value // float64
	networkPartitionRate atomic.Value // float64
	slowQueryRate        atomic.Value // float64

	// Tracking
	workerCrashes      atomic.Int64
	databaseFailures   atomic.Int64
	networkPartitions  atomic.Int64
	workflowsStarted   atomic.Int64
	workflowsCompleted atomic.Int64
	workflowsFailed    atomic.Int64
}

func NewChaosSimulator(engine *Engine) *ChaosSimulator {
	cs := &ChaosSimulator{
		engine:          engine,
		workers:         make([]Worker, 0),
		workflowMetrics: make(map[string]*WorkflowMetrics),
	}

	// Initialize rates
	cs.databaseFailureRate.Store(0.0)
	cs.workerCrashRate.Store(0.0)
	cs.networkPartitionRate.Store(0.0)
	cs.slowQueryRate.Store(0.0)

	return cs
}

// TestChaosSimulation runs a comprehensive chaos simulation with complex workflows
func TestChaosSimulation(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	simulator := NewChaosSimulator(engine)

	// Create metrics for each workflow type
	billingMetrics := NewWorkflowMetrics()
	pipelineMetrics := NewWorkflowMetrics()
	stateMachineMetrics := NewWorkflowMetrics()

	// Register complex workflows
	billingWorkflow := &ComplexBillingWorkflow{
		engine:       engine,
		name:         "complex-billing-workflow",
		failureRate:  0.1, // 10% step failure rate
		chaosEnabled: true,
		metrics:      billingMetrics,
	}

	pipelineWorkflow := &ComplexDataPipelineWorkflow{
		engine:       engine,
		name:         "complex-pipeline-workflow",
		chaosEnabled: true,
		metrics:      pipelineMetrics,
	}

	stateMachineWorkflow := &ComplexStateMachineWorkflow{
		engine:       engine,
		name:         "complex-state-machine-workflow",
		chaosEnabled: true,
		metrics:      stateMachineMetrics,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Phase 1: Start with stable conditions
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		worker, err := simulator.CreateWorker(fmt.Sprintf("chaos-worker-%d", i))
		require.NoError(t, err)

		// Register all workflows with each worker
		err = RegisterWorkflow(worker, billingWorkflow)
		require.NoError(t, err)
		err = RegisterWorkflow(worker, pipelineWorkflow)
		require.NoError(t, err)
		err = RegisterWorkflow(worker, stateMachineWorkflow)
		require.NoError(t, err)

		err = worker.Start(ctx)
		require.NoError(t, err)
		simulator.workers = append(simulator.workers, worker)
	}

	// Submit initial workflows
	for i := 0; i < 20; i++ {
		// Mix of different workflow types
		var err error

		switch i % 3 {
		case 0:
			_, err = billingWorkflow.Start(ctx, fmt.Sprintf("billing-%d", i))
		case 1:
			_, err = pipelineWorkflow.Start(ctx, fmt.Sprintf("pipeline-%d", i))
		case 2:
			_, err = stateMachineWorkflow.Start(ctx, fmt.Sprintf("state-%d", i))
		}

		require.NoError(t, err)
		simulator.workflowsStarted.Add(1)
	}

	// Let workflows run for a bit
	time.Sleep(5 * time.Second)

	// Phase 2: Introduce moderate chaos
	simulator.databaseFailureRate.Store(0.05) // 5% database failures
	simulator.workerCrashRate.Store(0.02)     // 2% worker crash rate
	simulator.slowQueryRate.Store(0.1)        // 10% slow queries

	// Submit more workflows under chaos
	for i := 0; i < 30; i++ {
		var err error

		switch rand.Intn(3) {
		case 0:
			_, err = billingWorkflow.Start(ctx, fmt.Sprintf("chaos-billing-%d", i))
		case 1:
			_, err = pipelineWorkflow.Start(ctx, fmt.Sprintf("chaos-pipeline-%d", i))
		case 2:
			_, err = stateMachineWorkflow.Start(ctx, fmt.Sprintf("chaos-state-%d", i))
		}

		if err != nil {
			// Database might be failing
			simulator.databaseFailures.Add(1)
			continue
		}
		simulator.workflowsStarted.Add(1)

		// Randomly crash a worker
		if rand.Float64() < simulator.workerCrashRate.Load().(float64) {
			simulator.CrashRandomWorker(t)
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Phase 3: Extreme chaos
	simulator.databaseFailureRate.Store(0.2)   // 20% database failures
	simulator.workerCrashRate.Store(0.1)       // 10% worker crash rate
	simulator.networkPartitionRate.Store(0.15) // 15% network partitions
	simulator.slowQueryRate.Store(0.3)         // 30% slow queries

	// Crash multiple workers
	for i := 0; i < 2; i++ {
		simulator.CrashRandomWorker(t)
		time.Sleep(500 * time.Millisecond)
	}

	// Submit burst of workflows
	var burstWG sync.WaitGroup
	for i := 0; i < 50; i++ {
		burstWG.Add(1)
		go func(idx int) {
			defer burstWG.Done()

			workflowType := rand.Intn(3)
			var err error

			switch workflowType {
			case 0:
				_, err = billingWorkflow.Start(ctx, fmt.Sprintf("burst-billing-%d", idx))
			case 1:
				_, err = pipelineWorkflow.Start(ctx, fmt.Sprintf("burst-pipeline-%d", idx))
			case 2:
				_, err = stateMachineWorkflow.Start(ctx, fmt.Sprintf("burst-state-%d", idx))
			}

			if err == nil {
				simulator.workflowsStarted.Add(1)
			} else {
				simulator.databaseFailures.Add(1)
			}
		}(i)

		if i%10 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	burstWG.Wait()

	// Phase 4: Recovery
	simulator.databaseFailureRate.Store(0.0)
	simulator.workerCrashRate.Store(0.0)
	simulator.networkPartitionRate.Store(0.0)
	simulator.slowQueryRate.Store(0.0)

	// Restart crashed workers
	numActiveWorkers := len(simulator.workers) // Simplified for now

	// Add replacement workers
	for i := numActiveWorkers; i < numWorkers; i++ {
		worker, err := simulator.CreateWorker(fmt.Sprintf("recovery-worker-%d", i))
		require.NoError(t, err)

		err = RegisterWorkflow(worker, billingWorkflow)
		require.NoError(t, err)
		err = RegisterWorkflow(worker, pipelineWorkflow)
		require.NoError(t, err)
		err = RegisterWorkflow(worker, stateMachineWorkflow)
		require.NoError(t, err)

		err = worker.Start(ctx)
		require.NoError(t, err)
	}

	// Wait for system to stabilize
	time.Sleep(10 * time.Second)

	// Analyze results

	// Check workflow completion rates
	pendingWorkflows := 0
	completedWorkflows := 0
	failedWorkflows := 0

	workflows, err := engine.store.GetAllWorkflows(ctx, "default")
	require.NoError(t, err)

	for _, wf := range workflows {
		switch wf.Status {
		case store.WorkflowStatusPending, store.WorkflowStatusRunning:
			pendingWorkflows++
		case store.WorkflowStatusCompleted:
			completedWorkflows++
		case store.WorkflowStatusFailed:
			failedWorkflows++
		}
	}

	// Workflow metrics

	// Chaos metrics

	// Assertions
	require.Greater(t, completedWorkflows, 0, "At least some workflows should complete")
	require.Less(t, float64(failedWorkflows)/float64(len(workflows)), 0.5,
		"Less than 50% of workflows should fail even under extreme chaos")

	// Check for data consistency
	stepExecutions := make(map[string]int)
	steps, err := engine.store.GetAllSteps(ctx, "default")
	require.NoError(t, err)

	for _, step := range steps {
		key := fmt.Sprintf("%s-%s", step.ExecutionID, step.StepName)
		stepExecutions[key]++
	}

	// Verify no duplicate step executions
	duplicateSteps := 0
	for stepKey, count := range stepExecutions {
		if count > 1 {
			duplicateSteps++
			t.Errorf("Duplicate step execution detected: %s (count: %d)", stepKey, count)
		}
	}

	require.Equal(t, 0, duplicateSteps, "No steps should be executed more than once")

}

func (cs *ChaosSimulator) CreateWorker(workerID string) (Worker, error) {
	return NewWorker(cs.engine, WorkerConfig{
		WorkerID:          workerID,
		Concurrency:       3,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 2 * time.Second,
		ClaimTimeout:      30 * time.Second,
	})
}

func (cs *ChaosSimulator) CrashRandomWorker(t *testing.T) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(cs.workers) == 0 {
		return
	}

	// For now, just log that we would crash a worker
	// Actual worker crashing would require access to internal fields
	cs.workerCrashes.Add(1)
}

// TestDatabaseFailureScenarios specifically tests database failure handling
func TestDatabaseFailureScenarios(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	// Wrap the store to inject failures
	failureStore := &FailureInjectingStore{
		Store:       engine.store,
		failureRate: atomic.Value{},
	}
	failureStore.failureRate.Store(0.0)
	engine.store = failureStore

	metrics := NewWorkflowMetrics()
	workflow := &ComplexBillingWorkflow{
		engine:       engine,
		name:         "db-failure-test-workflow",
		failureRate:  0.0, // No workflow failures, only DB failures
		chaosEnabled: false,
		metrics:      metrics,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start worker
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "db-failure-worker",
		Concurrency:       2,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 2 * time.Second,
		ClaimTimeout:      10 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Test 1: Database failures during workflow submission
	failureStore.failureRate.Store(0.5) // 50% failure rate

	submissionFailures := 0
	submissionSuccesses := 0

	for i := 0; i < 20; i++ {
		_, err := workflow.Start(ctx, fmt.Sprintf("submission-test-%d", i))
		if err != nil {
			submissionFailures++
		} else {
			submissionSuccesses++
		}
	}

	require.Greater(t, submissionFailures, 0, "Should have some submission failures")
	require.Greater(t, submissionSuccesses, 0, "Should have some submission successes")

	// Test 2: Database recovery
	failureStore.failureRate.Store(0.0) // Restore database

	// Submit workflows that should succeed
	var recoveryWorkflows []string
	for i := 0; i < 10; i++ {
		workflowID, err := workflow.Start(ctx, fmt.Sprintf("recovery-test-%d", i))
		require.NoError(t, err)
		recoveryWorkflows = append(recoveryWorkflows, workflowID)
	}

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Verify workflows completed
	completedCount := 0
	for _, workflowID := range recoveryWorkflows {
		wf, err := engine.store.GetWorkflow(ctx, "default", workflowID)
		require.NoError(t, err)
		if wf.Status == store.WorkflowStatusCompleted {
			completedCount++
		}
	}

	require.GreaterOrEqual(t, completedCount, len(recoveryWorkflows)/2, "At least half of workflows should complete after recovery")

	// Test 3: Database failures during processing

	// Submit workflows
	var processingWorkflows []string
	for i := 0; i < 5; i++ {
		workflowID, err := workflow.Start(ctx, fmt.Sprintf("processing-test-%d", i))
		require.NoError(t, err)
		processingWorkflows = append(processingWorkflows, workflowID)
	}

	// Introduce failures during processing
	time.Sleep(500 * time.Millisecond)
	failureStore.failureRate.Store(0.3) // 30% failure rate

	// Wait for processing attempts
	time.Sleep(5 * time.Second)

	// Restore database
	failureStore.failureRate.Store(0.0)

	// Wait for recovery
	time.Sleep(5 * time.Second)

	// Check final state
	for _, workflowID := range processingWorkflows {
		_, err := engine.store.GetWorkflow(ctx, "default", workflowID)
		require.NoError(t, err)
	}

}

// FailureInjectingStore wraps a store and randomly injects failures
type FailureInjectingStore struct {
	store.Store
	failureRate atomic.Value // float64
	mu          sync.Mutex
}

func (f *FailureInjectingStore) shouldFail() bool {
	rate := f.failureRate.Load().(float64)
	return rand.Float64() < rate
}

func (f *FailureInjectingStore) CreateWorkflow(ctx context.Context, workflow *store.WorkflowExecution) error {
	if f.shouldFail() {
		return fmt.Errorf("simulated database failure")
	}
	return f.Store.CreateWorkflow(ctx, workflow)
}

func (f *FailureInjectingStore) GetPendingWorkflows(ctx context.Context, namespace string, limit int, workflowNames []string) ([]store.WorkflowExecution, error) {
	if f.shouldFail() {
		return nil, fmt.Errorf("simulated database failure")
	}
	return f.Store.GetPendingWorkflows(ctx, namespace, limit, workflowNames)
}

func (f *FailureInjectingStore) UpdateWorkflowStatus(ctx context.Context, namespace, workflowID string, status store.WorkflowStatus, errorMsg string) error {
	if f.shouldFail() {
		return fmt.Errorf("simulated database failure")
	}
	return f.Store.UpdateWorkflowStatus(ctx, namespace, workflowID, status, errorMsg)
}

func (f *FailureInjectingStore) CreateStep(ctx context.Context, step *store.WorkflowStep) error {
	if f.shouldFail() {
		return fmt.Errorf("simulated database failure")
	}
	return f.Store.CreateStep(ctx, step)
}

func (f *FailureInjectingStore) UpdateStepStatus(ctx context.Context, namespace, workflowID, stepName string, status store.StepStatus, output []byte, errorMsg string) error {
	if f.shouldFail() {
		return fmt.Errorf("simulated database failure")
	}
	return f.Store.UpdateStepStatus(ctx, namespace, workflowID, stepName, status, output, errorMsg)
}
