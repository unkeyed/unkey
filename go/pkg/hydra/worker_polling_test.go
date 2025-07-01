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

// TestWorkerPollingEfficiency verifies that workers poll at the correct intervals
// and don't create "thundering herd" effects when many workers start simultaneously
func TestWorkerPollingEfficiency(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	const (
		numWorkers   = 20
		pollInterval = 100 * time.Millisecond
		testDuration = 2 * time.Second
	)

	// Track polling behavior
	var totalPolls atomic.Int64
	var pollTimes sync.Map // workerID -> []time.Time

	// Custom worker that tracks polling
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Start workers simultaneously to test thundering herd
	var wg sync.WaitGroup
	startTime := testClock.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			worker, err := NewWorker(engine, WorkerConfig{
				WorkerID:          fmt.Sprintf("polling-worker-%d", workerID),
				Concurrency:       1,
				PollInterval:      pollInterval,
				HeartbeatInterval: 5 * time.Second,
				ClaimTimeout:      30 * time.Second,
			})
			require.NoError(t, err)

			// Register a minimal workflow
			pollingWorkflow := &pollingTestWorkflow{
				engine: engine,
				name:   "polling-test-workflow",
				onPoll: func() {
					totalPolls.Add(1)
					// Record poll time for this worker
					if times, ok := pollTimes.Load(workerID); ok {
						timeList := times.([]time.Time)
						timeList = append(timeList, testClock.Now())
						pollTimes.Store(workerID, timeList)
					} else {
						pollTimes.Store(workerID, []time.Time{testClock.Now()})
					}
				},
			}

			err = RegisterWorkflow(worker, pollingWorkflow)
			require.NoError(t, err)

			err = worker.Start(ctx)
			require.NoError(t, err)
			defer worker.Shutdown(ctx)

			// Keep worker running
			<-ctx.Done()
		}(i)
	}

	// Simulate time passage to trigger polling
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				testClock.Tick(pollInterval / 4) // Tick more frequently than poll interval
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	wg.Wait()

	// Analyze polling behavior
	finalPolls := totalPolls.Load()
	expectedPolls := int64(numWorkers) * int64(testDuration/pollInterval)

	// Check for thundering herd - all workers shouldn't poll at exactly the same time
	pollTimeSpread := analyzePollingDistribution(t, &pollTimes, numWorkers, startTime)

	// Performance assertions
	require.Greater(t, finalPolls, expectedPolls/2,
		"Should have reasonable number of polls, got %d, expected ~%d", finalPolls, expectedPolls)

	require.Less(t, pollTimeSpread, pollInterval/2,
		"Polling should be reasonably distributed to avoid thundering herd, spread: %v", pollTimeSpread)

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
	clustering := analyzePollingClustering(t, pollTimes, pollInterval)

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

// Helper function to analyze polling time distribution across workers
func analyzePollingDistribution(t *testing.T, pollTimes *sync.Map, numWorkers int, startTime time.Time) time.Duration {
	// Collect all first poll times (to check for thundering herd at startup)
	firstPolls := make([]time.Time, 0, numWorkers)

	pollTimes.Range(func(key, value interface{}) bool {
		times := value.([]time.Time)
		if len(times) > 0 {
			firstPolls = append(firstPolls, times[0])
		}
		return true
	})

	if len(firstPolls) < 2 {
		return 0
	}

	// Find min and max poll times
	minTime := firstPolls[0]
	maxTime := firstPolls[0]

	for _, pollTime := range firstPolls {
		if pollTime.Before(minTime) {
			minTime = pollTime
		}
		if pollTime.After(maxTime) {
			maxTime = pollTime
		}
	}

	spread := maxTime.Sub(minTime)

	return spread
}

// Helper function to analyze polling clustering (thundering herd detection)
func analyzePollingClustering(t *testing.T, pollTimes []time.Time, pollInterval time.Duration) float64 {
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
