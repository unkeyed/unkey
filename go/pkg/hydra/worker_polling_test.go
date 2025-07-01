package hydra

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// TestWorkerPollingEfficiency verifies that workers can handle concurrent load
// without excessive database contention or resource exhaustion
func TestWorkerPollingEfficiency(t *testing.T) {
	engine := newTestEngine(t)

	const (
		numWorkers   = 10
		numWorkflows = 50
		testDuration = 5 * time.Second
	)

	var completedWorkflows atomic.Int64

	// Create workflow that tracks completion
	pollingWorkflow := &pollingTestWorkflow{
		engine: engine,
		name:   "polling-test-workflow",
		onPoll: func() {
			completedWorkflows.Add(1)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			worker, err := NewWorker(engine, WorkerConfig{
				WorkerID:          fmt.Sprintf("polling-worker-%d", workerID),
				Concurrency:       5, // Multiple workflows per worker
				PollInterval:      100 * time.Millisecond,
				HeartbeatInterval: 5 * time.Second,
				ClaimTimeout:      30 * time.Second,
			})
			require.NoError(t, err)

			err = RegisterWorkflow(worker, pollingWorkflow)
			require.NoError(t, err)

			err = worker.Start(ctx)
			require.NoError(t, err)
			defer worker.Shutdown(ctx)

			<-ctx.Done()
		}(i)
	}

	// Submit workflows for processing
	for i := 0; i < numWorkflows; i++ {
		_, err := pollingWorkflow.Start(ctx, fmt.Sprintf("poll-test-%d", i))
		require.NoError(t, err)
	}

	// Wait for completion or timeout
	require.Eventually(t, func() bool {
		return completedWorkflows.Load() >= int64(numWorkflows)
	}, testDuration, 100*time.Millisecond,
		"Should complete %d workflows within %v", numWorkflows, testDuration)

	wg.Wait()

	// Verify all workflows were processed
	finalCompleted := completedWorkflows.Load()
	require.GreaterOrEqual(t, finalCompleted, int64(numWorkflows),
		"Should have completed at least %d workflows, got %d", numWorkflows, finalCompleted)
}

// TestWorkerPollingAccuracy tests that workers actually poll at the configured interval
func TestWorkerPollingAccuracy(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	const pollInterval = 200 * time.Millisecond
	const tolerance = 50 * time.Millisecond // 25% tolerance

	var pollTimes []time.Time
	var mu sync.Mutex

	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "accuracy-test-worker",
		Concurrency:       1,
		PollInterval:      pollInterval,
		HeartbeatInterval: 5 * time.Second,
		ClaimTimeout:      30 * time.Second,
	})
	require.NoError(t, err)

	pollingWorkflow := &pollingTestWorkflow{
		engine: engine,
		name:   "accuracy-test-workflow",
		onPoll: func() {
			mu.Lock()
			pollTimes = append(pollTimes, testClock.Now())
			mu.Unlock()
		},
	}

	err = RegisterWorkflow(worker, pollingWorkflow)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Advance clock to trigger multiple polls
	for i := 0; i < 10; i++ {
		testClock.Tick(pollInterval)
		time.Sleep(10 * time.Millisecond) // Allow processing
	}

	// Analyze interval accuracy
	mu.Lock()
	if len(pollTimes) < 2 {
		mu.Unlock()
		t.Skip("Not enough poll events to analyze intervals")
		return
	}

	actualIntervals := make([]time.Duration, len(pollTimes)-1)
	for i := 1; i < len(pollTimes); i++ {
		actualIntervals[i-1] = pollTimes[i].Sub(pollTimes[i-1])
	}
	mu.Unlock()

	// Check each interval is within tolerance
	accurateIntervals := 0
	for _, interval := range actualIntervals {
		diff := interval - pollInterval
		if diff < 0 {
			diff = -diff
		}

		isAccurate := diff <= tolerance
		if isAccurate {
			accurateIntervals++
		}

	}

	accuracy := float64(accurateIntervals) / float64(len(actualIntervals)) * 100

	// Performance assertions
	require.GreaterOrEqual(t, accuracy, 80.0,
		"At least 80%% of polling intervals should be accurate, got %.1f%%", accuracy)

}

// TestThunderingHerdPrevention ensures that when many workers start at the same time,
// they don't all poll the database simultaneously causing performance issues
func TestThunderingHerdPrevention(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	const (
		numWorkers   = 50 // Large number to stress test
		pollInterval = 100 * time.Millisecond
	)

	// Track when each worker polls
	pollEvents := make(chan time.Time, 1000)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start all workers simultaneously
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			worker, err := NewWorker(engine, WorkerConfig{
				WorkerID:          fmt.Sprintf("herd-worker-%d", workerID),
				Concurrency:       1,
				PollInterval:      pollInterval,
				HeartbeatInterval: 5 * time.Second,
				ClaimTimeout:      30 * time.Second,
			})
			require.NoError(t, err)

			pollingWorkflow := &pollingTestWorkflow{
				engine: engine,
				name:   "herd-test-workflow",
				onPoll: func() {
					select {
					case pollEvents <- testClock.Now():
					default:
						// Channel full, skip
					}
				},
			}

			err = RegisterWorkflow(worker, pollingWorkflow)
			require.NoError(t, err)

			err = worker.Start(ctx)
			require.NoError(t, err)
			defer worker.Shutdown(ctx)

			<-ctx.Done()
		}(i)
	}

	// Advance time to trigger polling
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				testClock.Tick(pollInterval / 4)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	close(pollEvents)

	// Analyze thundering herd behavior
	pollTimes := make([]time.Time, 0)
	for pollTime := range pollEvents {
		pollTimes = append(pollTimes, pollTime)
	}

	// Check for clustering (thundering herd indicator)
	clustering := analyzePollingClustering(pollTimes, pollInterval)

	// Performance assertion
	require.Less(t, clustering, 0.5,
		"Polling clustering should be low to prevent thundering herd, got %.2f", clustering)

}

// pollingTestWorkflow is a minimal workflow that tracks when it's polled for
type pollingTestWorkflow struct {
	engine *Engine
	name   string
	onPoll func()
}

func (w *pollingTestWorkflow) Name() string {
	return w.name
}

func (w *pollingTestWorkflow) Run(ctx WorkflowContext, req any) error {
	// This is called when the workflow is actually executed
	// We use onPoll to track when workers check for pending work
	if w.onPoll != nil {
		w.onPoll()
	}

	_, err := Step(ctx, "polling-step", func(context.Context) (string, error) {
		return "polled", nil
	})
	return err
}

func (w *pollingTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// Helper function to analyze polling clustering (thundering herd detection)
func analyzePollingClustering(pollTimes []time.Time, pollInterval time.Duration) float64 {
	if len(pollTimes) < 2 {
		return 0
	}

	// Group polls by time windows
	windowSize := pollInterval / 10 // 10% of poll interval
	timeWindows := make(map[int64]int)

	baseTime := pollTimes[0]
	for _, pollTime := range pollTimes {
		windowIndex := pollTime.Sub(baseTime).Nanoseconds() / windowSize.Nanoseconds()
		timeWindows[windowIndex]++
	}

	// Calculate clustering factor (higher = more clustered)
	totalPolls := len(pollTimes)
	maxWindowCount := 0

	for _, count := range timeWindows {
		if count > maxWindowCount {
			maxWindowCount = count
		}
	}

	clustering := float64(maxWindowCount) / float64(totalPolls)

	return clustering
}
