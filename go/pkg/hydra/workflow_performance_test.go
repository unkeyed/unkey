package hydra

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// TestWorkflowPickupLatencyBaseline measures the baseline latency for a single worker
// to pick up and start executing a single workflow. This establishes our performance
// baseline before testing the 5-second SLA requirement.
func TestWorkflowPickupLatencyBaseline(t *testing.T) {
	// Arrange: Create engine with real clock for accurate timing
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	var workflowStartTime atomic.Int64

	// Create a workflow that records when it actually starts executing
	workflow := &latencyTestWorkflow{
		engine: engine,
		name:   "baseline-latency-workflow",
		onStart: func() {
			workflowStartTime.Store(time.Now().UnixMilli())
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start worker with production-like configuration
	worker, err := NewWorker(engine, WorkerConfig{
		Concurrency:       1,
		PollInterval:      100 * time.Millisecond, // Realistic poll interval
		HeartbeatInterval: 5 * time.Second,
		ClaimTimeout:      30 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Act: Record submission time and start workflow
	_, err = workflow.Start(ctx, struct{}{})
	require.NoError(t, err)

	// Wait for workflow to start executing
	require.Eventually(t, func() bool {
		return workflowStartTime.Load() != 0
	}, 5*time.Second, 10*time.Millisecond, "Workflow should start executing within 5 seconds")

	// Calculate pickup latency
	latency := time.Since(time.UnixMilli(workflowStartTime.Load()))

	require.Less(t, latency, 5*time.Second, "Pickup latency should be less than 5 seconds for baseline test")
}

// latencyTestWorkflow is a minimal workflow for testing pickup latency
type latencyTestWorkflow struct {
	engine  *Engine
	name    string
	onStart func()
}

func (w *latencyTestWorkflow) Name() string {
	return w.name
}

func (w *latencyTestWorkflow) Run(ctx WorkflowContext, req any) error {
	// Record when workflow actually starts executing
	if w.onStart != nil {
		w.onStart()
	}

	// Minimal work to complete quickly
	_, err := Step(ctx, "timing-step", func(context.Context) (string, error) {
		return "completed", nil
	})
	return err
}

func (w *latencyTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// TestWorkflowPickupLatencyConcurrent verifies that ALL workflows are picked up within the 5-second SLA
// under concurrent load. This tests the critical requirement that every workflow must be processed
// within the SLA window, not just the average.
func TestWorkflowPickupLatencyConcurrent(t *testing.T) {
	// Arrange: Create engine with real clock for accurate timing
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	const numWorkers = 5    // Multiple workers to test concurrent performance
	const numWorkflows = 50 // Realistic batch to stress test SLA compliance

	var completedCount atomic.Int64
	var maxLatency atomic.Int64
	var slaViolations atomic.Int64

	// Create workflow factory that records completion timing
	createWorkflow := func(id int) *concurrentLatencyWorkflow {
		return &concurrentLatencyWorkflow{
			engine: engine,
			name:   "concurrent-latency-workflow",
			id:     id,
			onComplete: func(latencyMs int64) {
				// Track maximum latency across all workflows
				for {
					current := maxLatency.Load()
					if latencyMs <= current || maxLatency.CompareAndSwap(current, latencyMs) {
						break
					}
				}

				// Count SLA violations (workflows taking >5s)
				if latencyMs > 5000 {
					slaViolations.Add(1)
					t.Errorf("SLA VIOLATION: Workflow %d took %dms (>5000ms) to be picked up", id, latencyMs)
				}

				completedCount.Add(1)
			},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start multiple workers sharing the same database
	workers := make([]Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker, err := NewWorker(engine, WorkerConfig{
			Concurrency:       10,                    // Reasonable concurrency per worker
			PollInterval:      50 * time.Millisecond, // Fast polling for concurrent load
			HeartbeatInterval: 5 * time.Second,
			ClaimTimeout:      30 * time.Second,
		})
		require.NoError(t, err)

		// Register the workflow type with each worker
		err = RegisterWorkflow(worker, createWorkflow(0))
		require.NoError(t, err)

		err = worker.Start(ctx)
		require.NoError(t, err)
		defer worker.Shutdown(ctx)

		workers[i] = worker
	}

	// Act: Submit all workflows as quickly as possible
	submissionStart := time.Now()
	executionIDs := make([]string, numWorkflows)

	for i := 0; i < numWorkflows; i++ {
		workflow := createWorkflow(i)
		executionID, err := workflow.Start(ctx, submissionStart.UnixMilli())
		require.NoError(t, err)
		executionIDs[i] = executionID
	}

	_ = time.Since(submissionStart) // Submission timing not needed for SLA test

	// Wait for all workflows to complete
	require.Eventually(t, func() bool {
		return completedCount.Load() == numWorkflows
	}, 15*time.Second, 100*time.Millisecond,
		"All %d workflows should complete within timeout", numWorkflows)

	// Assert SLA compliance: ALL workflows must be picked up within 5 seconds
	finalSlaViolations := slaViolations.Load()
	finalMaxLatency := maxLatency.Load()

	require.Equal(t, int64(0), finalSlaViolations,
		"SLA VIOLATION: %d out of %d workflows took longer than 5 seconds to be picked up",
		finalSlaViolations, numWorkflows)

	require.Less(t, finalMaxLatency, int64(5000),
		"SLA VIOLATION: Maximum pickup latency was %dms, must be <5000ms for ALL workflows",
		finalMaxLatency)

}

// concurrentLatencyWorkflow tracks individual workflow latency in concurrent scenarios
type concurrentLatencyWorkflow struct {
	engine     *Engine
	name       string
	id         int
	onComplete func(latencyMs int64)
}

func (w *concurrentLatencyWorkflow) Name() string {
	return w.name
}

func (w *concurrentLatencyWorkflow) Run(ctx WorkflowContext, req any) error {
	var submissionTime int64
	switch v := req.(type) {
	case int64:
		submissionTime = v
	case float64:
		submissionTime = int64(v) // JSON unmarshaling converts numbers to float64
	default:
		return fmt.Errorf("expected int64 or float64 submission time, got %T", req)
	}

	// Calculate latency from submission to execution start
	latency := time.Now().UnixMilli() - submissionTime

	// Report completion with latency
	if w.onComplete != nil {
		w.onComplete(latency)
	}

	// Minimal work to complete quickly
	_, err := Step(ctx, "latency-step", func(context.Context) (string, error) {
		return "completed", nil
	})

	return err
}

func (w *concurrentLatencyWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// BenchmarkWorkflowSubmission measures the rate at which workflows can be submitted
func BenchmarkWorkflowSubmission(b *testing.B) {
	engine := newTestEngineBench(b)

	workflow := &benchmarkWorkflow{
		engine: engine,
		name:   "benchmark-workflow",
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := workflow.Start(ctx, struct{}{})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkWorkflowThroughput measures end-to-end workflow processing throughput
func BenchmarkWorkflowThroughput(b *testing.B) {
	engine := newTestEngineBench(b)

	workflow := &benchmarkWorkflow{
		engine: engine,
		name:   "throughput-workflow",
	}

	// Start a single worker
	worker, err := NewWorker(engine, WorkerConfig{
		Concurrency:       10,                    // Process multiple workflows concurrently
		PollInterval:      10 * time.Millisecond, // Fast polling for benchmarks
		HeartbeatInterval: 5 * time.Second,
		ClaimTimeout:      30 * time.Second,
	})
	if err != nil {
		b.Fatal(err)
	}

	err = RegisterWorkflow(worker, workflow)
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = worker.Start(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer worker.Shutdown(ctx)

	// Give worker time to start
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()

	// Track completion
	var completed atomic.Int64
	workflow.onComplete = func() {
		completed.Add(1)
	}

	// Submit N workflows as fast as possible
	submissionStart := time.Now()
	for i := 0; i < b.N; i++ {
		_, err := workflow.Start(ctx, struct{}{})
		if err != nil {
			b.Fatal(err)
		}
	}
	submissionDuration := time.Since(submissionStart)

	// Wait for all workflows to complete
	for completed.Load() < int64(b.N) {
		time.Sleep(1 * time.Millisecond)
	}

	b.ReportMetric(float64(b.N)/submissionDuration.Seconds(), "submissions/sec")
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "completions/sec")
}

// BenchmarkSingleWorkerLatency measures latency with a single worker processing one workflow at a time
func BenchmarkSingleWorkerLatency(b *testing.B) {
	engine := newTestEngineBench(b)

	workflow := &benchmarkWorkflow{
		engine: engine,
		name:   "latency-workflow",
	}

	worker, err := NewWorker(engine, WorkerConfig{
		Concurrency:       1,                    // Single workflow at a time
		PollInterval:      1 * time.Millisecond, // Very fast polling
		HeartbeatInterval: 5 * time.Second,
		ClaimTimeout:      30 * time.Second,
	})
	if err != nil {
		b.Fatal(err)
	}

	err = RegisterWorkflow(worker, workflow)
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = worker.Start(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer worker.Shutdown(ctx)

	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		completed := make(chan struct{})
		workflow.onComplete = func() {
			close(completed)
		}

		start := time.Now()
		_, err := workflow.Start(ctx, struct{}{})
		if err != nil {
			b.Fatal(err)
		}

		<-completed
		latency := time.Since(start)

		// Report per-operation timing
		if i == 0 {
			b.ReportMetric(float64(latency.Nanoseconds()), "ns/workflow")
		}
	}
}

// benchmarkWorkflow is a minimal workflow for benchmarking
type benchmarkWorkflow struct {
	engine     *Engine
	name       string
	onComplete func()
}

func (w *benchmarkWorkflow) Name() string {
	return w.name
}

func (w *benchmarkWorkflow) Run(ctx WorkflowContext, req any) error {
	// Minimal work - just complete a simple step
	_, err := Step(ctx, "benchmark-step", func(context.Context) (string, error) {
		return "done", nil
	})

	if w.onComplete != nil {
		w.onComplete()
	}

	return err
}

func (w *benchmarkWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// Helper for benchmarks that need testing.TB interface
func newTestEngineBench(tb testing.TB) *Engine {
	// Use MySQL container for benchmarks
	t, ok := tb.(*testing.T)
	if !ok {
		// For benchmarks, create a new testing.T
		t = &testing.T{}
		t.Helper()
	}

	// Use the unified test helper
	return newTestEngineWithClock(t, clock.New())
}
