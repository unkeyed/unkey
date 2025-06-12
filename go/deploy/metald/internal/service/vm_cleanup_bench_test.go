package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
)

// Mock backend for cleanup benchmarks
type mockCleanupBackend struct {
	deleteLatency    time.Duration
	failureRate      float64 // 0.0 = never fail, 1.0 = always fail
	callCount        int64
	concurrentCalls  int64
	maxConcurrent    int64
}

func (m *mockCleanupBackend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	return "test-vm", nil
}

func (m *mockCleanupBackend) DeleteVM(ctx context.Context, vmID string) error {
	// Track concurrent calls
	current := atomic.AddInt64(&m.concurrentCalls, 1)
	defer atomic.AddInt64(&m.concurrentCalls, -1)
	
	// Update max concurrent if needed
	for {
		max := atomic.LoadInt64(&m.maxConcurrent)
		if current <= max || atomic.CompareAndSwapInt64(&m.maxConcurrent, max, current) {
			break
		}
	}
	
	// Increment total call count
	atomic.AddInt64(&m.callCount, 1)
	
	// Simulate latency
	if m.deleteLatency > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.deleteLatency):
		}
	}
	
	// Simulate failure rate
	if m.failureRate > 0 {
		callNum := atomic.LoadInt64(&m.callCount)
		if float64(callNum%100) < m.failureRate*100 {
			return errors.New("simulated backend failure")
		}
	}
	
	return nil
}

func (m *mockCleanupBackend) BootVM(ctx context.Context, vmID string) error   { return nil }
func (m *mockCleanupBackend) ShutdownVM(ctx context.Context, vmID string) error { return nil }
func (m *mockCleanupBackend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeout int32) error { return nil }
func (m *mockCleanupBackend) PauseVM(ctx context.Context, vmID string) error  { return nil }
func (m *mockCleanupBackend) ResumeVM(ctx context.Context, vmID string) error { return nil }
func (m *mockCleanupBackend) RebootVM(ctx context.Context, vmID string) error { return nil }
func (m *mockCleanupBackend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) { return nil, nil }
func (m *mockCleanupBackend) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) { return nil, nil }
func (m *mockCleanupBackend) Ping(ctx context.Context) error { return nil }

func (m *mockCleanupBackend) GetCallCount() int64     { return atomic.LoadInt64(&m.callCount) }
func (m *mockCleanupBackend) GetMaxConcurrent() int64 { return atomic.LoadInt64(&m.maxConcurrent) }
func (m *mockCleanupBackend) Reset() {
	atomic.StoreInt64(&m.callCount, 0)
	atomic.StoreInt64(&m.concurrentCalls, 0)
	atomic.StoreInt64(&m.maxConcurrent, 0)
}

// createBenchmarkVMService creates a VM service for benchmarking cleanup operations
func createBenchmarkVMService(backend types.Backend) *VMService {
	// Use a discarding logger for benchmarks to avoid I/O overhead
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors to reduce noise
	}))
	
	// Use nil for optional components in benchmarks
	return &VMService{
		backend:          backend,
		logger:           logger,
		metricsCollector: nil,
		vmMetrics:        nil,
		vmRepo:          nil,
	}
}

// BenchmarkCleanupSuccess tests successful cleanup performance
func BenchmarkCleanupSuccess(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 10 * time.Millisecond, // Realistic backend latency
		failureRate:   0.0,                   // No failures
	}
	service := createBenchmarkVMService(backend)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		vmID := fmt.Sprintf("vm-%d", i)
		success := service.performVMCleanup(ctx, vmID, "benchmark_test")
		if !success {
			b.Errorf("cleanup failed unexpectedly for vm %s", vmID)
		}
	}
	
	b.ReportMetric(float64(backend.GetCallCount()), "backend_calls")
	b.ReportMetric(float64(backend.GetMaxConcurrent()), "max_concurrent")
}

// BenchmarkCleanupWithRetries tests cleanup performance when retries are needed
func BenchmarkCleanupWithRetries(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 5 * time.Millisecond,
		failureRate:   0.4, // 40% failure rate to trigger retries
	}
	service := createBenchmarkVMService(backend)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		vmID := fmt.Sprintf("vm-%d", i)
		service.performVMCleanup(ctx, vmID, "benchmark_test_retries")
	}
	
	b.ReportMetric(float64(backend.GetCallCount()), "backend_calls")
	b.ReportMetric(float64(backend.GetMaxConcurrent()), "max_concurrent")
}

// BenchmarkCleanupConcurrent tests concurrent cleanup performance
func BenchmarkCleanupConcurrent(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 20 * time.Millisecond,
		failureRate:   0.1, // 10% failure rate
	}
	service := createBenchmarkVMService(backend)
	
	concurrencyLevels := []int{1, 10, 50, 100, 200}
	
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			backend.Reset()
			
			b.ResetTimer()
			b.ReportAllocs()
			
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					ctx := context.Background()
					vmID := fmt.Sprintf("vm-%d-%d", concurrency, i)
					service.performVMCleanup(ctx, vmID, "benchmark_concurrent")
					i++
				}
			})
			
			b.ReportMetric(float64(backend.GetCallCount()), "backend_calls")
			b.ReportMetric(float64(backend.GetMaxConcurrent()), "max_concurrent")
		})
	}
}

// BenchmarkCleanupSlowBackend tests cleanup with slow backend responses
func BenchmarkCleanupSlowBackend(b *testing.B) {
	latencies := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}
	
	for _, latency := range latencies {
		b.Run(fmt.Sprintf("latency-%s", latency), func(b *testing.B) {
			backend := &mockCleanupBackend{
				deleteLatency: latency,
				failureRate:   0.0,
			}
			service := createBenchmarkVMService(backend)
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				vmID := fmt.Sprintf("vm-%d", i)
				service.performVMCleanup(ctx, vmID, "benchmark_slow_backend")
			}
			
			b.ReportMetric(float64(backend.GetCallCount()), "backend_calls")
		})
	}
}

// BenchmarkCleanupContextCancellation tests cleanup behavior with context cancellation
func BenchmarkCleanupContextCancellation(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 100 * time.Millisecond,
		failureRate:   0.0,
	}
	service := createBenchmarkVMService(backend)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Create context that cancels after 50ms (before operation completes)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		vmID := fmt.Sprintf("vm-%d", i)
		
		// This should still succeed due to grace period context
		success := service.performVMCleanup(ctx, vmID, "benchmark_cancellation")
		cancel()
		
		// Even with cancelled context, cleanup should succeed due to grace period
		if !success {
			b.Errorf("cleanup failed for vm %s with cancelled context", vmID)
		}
	}
	
	b.ReportMetric(float64(backend.GetCallCount()), "backend_calls")
}

// BenchmarkCleanupMemoryUsage measures memory allocation patterns
func BenchmarkCleanupMemoryUsage(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 1 * time.Millisecond,
		failureRate:   0.2,
	}
	service := createBenchmarkVMService(backend)
	
	// Pre-allocate VM IDs to avoid allocation during benchmark
	vmIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		vmIDs[i] = fmt.Sprintf("vm-%d", i)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		service.performVMCleanup(ctx, vmIDs[i], "benchmark_memory")
	}
}

// BenchmarkCleanupStressTest simulates high-load cleanup scenarios
func BenchmarkCleanupStressTest(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 25 * time.Millisecond,
		failureRate:   0.15, // 15% failure rate
	}
	service := createBenchmarkVMService(backend)
	
	// Simulate burst cleanup scenarios
	burstSizes := []int{10, 50, 100, 500}
	
	for _, burstSize := range burstSizes {
		b.Run(fmt.Sprintf("burst-%d", burstSize), func(b *testing.B) {
			backend.Reset()
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				startTime := time.Now()
				
				// Launch burst of concurrent cleanups
				for j := 0; j < burstSize; j++ {
					wg.Add(1)
					go func(vmIndex int) {
						defer wg.Done()
						ctx := context.Background()
						vmID := fmt.Sprintf("vm-%d-%d", i, vmIndex)
						service.performVMCleanup(ctx, vmID, "benchmark_stress")
					}(j)
				}
				
				wg.Wait()
				
				// Report burst completion time
				burstDuration := time.Since(startTime)
				b.ReportMetric(float64(burstDuration.Nanoseconds()), "burst_duration_ns")
			}
			
			b.ReportMetric(float64(backend.GetCallCount()), "backend_calls")
			b.ReportMetric(float64(backend.GetMaxConcurrent()), "max_concurrent")
		})
	}
}

// BenchmarkCleanupFailureRecovery tests cleanup behavior under total backend failure
func BenchmarkCleanupFailureRecovery(b *testing.B) {
	backend := &mockCleanupBackend{
		deleteLatency: 10 * time.Millisecond,
		failureRate:   1.0, // 100% failure rate
	}
	service := createBenchmarkVMService(backend)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		vmID := fmt.Sprintf("vm-%d", i)
		success := service.performVMCleanup(ctx, vmID, "benchmark_failure")
		
		// Should fail since backend always fails
		if success {
			b.Errorf("cleanup succeeded unexpectedly for vm %s", vmID)
		}
	}
	
	// Should see exactly 3 attempts per VM (3 retries)
	expectedCalls := int64(b.N * 3)
	actualCalls := backend.GetCallCount()
	if actualCalls != expectedCalls {
		b.Errorf("expected %d backend calls, got %d", expectedCalls, actualCalls)
	}
	
	b.ReportMetric(float64(actualCalls), "backend_calls")
}