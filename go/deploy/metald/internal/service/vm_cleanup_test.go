package service

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestPerformVMCleanupBasic validates the basic cleanup functionality
func TestPerformVMCleanupBasic(t *testing.T) {
	backend := &mockCleanupBackend{
		deleteLatency: 1 * time.Millisecond,
		failureRate:   0.0,
	}
	service := createBenchmarkVMService(backend)

	ctx := context.Background()
	success := service.performVMCleanup(ctx, "test-vm", "test_reason")

	if !success {
		t.Error("cleanup should have succeeded")
	}

	if backend.GetCallCount() != 1 {
		t.Errorf("expected 1 backend call, got %d", backend.GetCallCount())
	}
}

// TestPerformVMCleanupWithRetries validates retry logic
func TestPerformVMCleanupWithRetries(t *testing.T) {
	backend := &mockCleanupBackend{
		deleteLatency: 1 * time.Millisecond,
		failureRate:   0.3, // Lower failure rate to ensure eventual success
	}
	service := createBenchmarkVMService(backend)

	ctx := context.Background()

	// Run multiple attempts to test retry logic
	successCount := 0
	totalAttempts := 10

	for i := 0; i < totalAttempts; i++ {
		backend.Reset()
		vmID := fmt.Sprintf("test-vm-%d", i)
		success := service.performVMCleanup(ctx, vmID, "test_retries")
		if success {
			successCount++
		}
	}

	// With 30% failure rate and 3 retries, we should see high success rate
	successRate := float64(successCount) / float64(totalAttempts)
	if successRate < 0.8 { // Expect at least 80% success
		t.Errorf("success rate too low: %.2f (expected >= 0.8)", successRate)
	}

	t.Logf("Retry test: %d/%d succeeded (%.1f%%)", successCount, totalAttempts, successRate*100)
}

// TestPerformVMCleanupFailure validates failure handling
func TestPerformVMCleanupFailure(t *testing.T) {
	backend := &mockCleanupBackend{
		deleteLatency: 1 * time.Millisecond,
		failureRate:   1.0, // Always fail
	}
	service := createBenchmarkVMService(backend)

	ctx := context.Background()
	success := service.performVMCleanup(ctx, "test-vm", "test_failure")

	// Should fail after all retries
	if success {
		t.Error("cleanup should have failed")
	}

	// Should have made 3 attempts
	if backend.GetCallCount() != 3 {
		t.Errorf("expected 3 backend calls, got %d", backend.GetCallCount())
	}
}

// TestPerformVMCleanupGracePeriod validates context handling
func TestPerformVMCleanupGracePeriod(t *testing.T) {
	backend := &mockCleanupBackend{
		deleteLatency: 50 * time.Millisecond, // Longer than cancellation
		failureRate:   0.0,
	}
	service := createBenchmarkVMService(backend)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	success := service.performVMCleanup(ctx, "test-vm", "test_grace_period")

	// Should still succeed due to grace period context
	if !success {
		t.Error("cleanup should have succeeded despite context cancellation")
	}

	if backend.GetCallCount() != 1 {
		t.Errorf("expected 1 backend call, got %d", backend.GetCallCount())
	}
}

// TestPerformVMCleanupConcurrent validates concurrent cleanup safety
func TestPerformVMCleanupConcurrent(t *testing.T) {
	backend := &mockCleanupBackend{
		deleteLatency: 10 * time.Millisecond,
		failureRate:   0.1,
	}
	service := createBenchmarkVMService(backend)

	const numGoroutines = 20
	const cleanupPerGoroutine = 5

	results := make(chan bool, numGoroutines*cleanupPerGoroutine)

	// Launch concurrent cleanups
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < cleanupPerGoroutine; j++ {
				ctx := context.Background()
				vmID := fmt.Sprintf("vm-%d-%d", goroutineID, j)
				success := service.performVMCleanup(ctx, vmID, "test_concurrent")
				results <- success
			}
		}(i)
	}

	// Collect results
	successCount := 0
	totalCount := 0
	for totalCount < numGoroutines*cleanupPerGoroutine {
		success := <-results
		if success {
			successCount++
		}
		totalCount++
	}

	// Should have high success rate despite some failures
	successRate := float64(successCount) / float64(totalCount)
	if successRate < 0.8 { // Allow for some failures due to 10% failure rate
		t.Errorf("success rate too low: %.2f", successRate)
	}

	t.Logf("Concurrent cleanup test: %d/%d succeeded (%.1f%%), max concurrent: %d",
		successCount, totalCount, successRate*100, backend.GetMaxConcurrent())
}
