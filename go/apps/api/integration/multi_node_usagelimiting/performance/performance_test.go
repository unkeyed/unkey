//go:build integration

package multi_node_usagelimiting

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// TestUsageLimitPerformance tests the performance of usage limiting
func TestUsageLimitPerformance(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	testCases := []struct {
		name      string
		nodeCount int
		credits   int64
		cost      int64
	}{
		{"SingleNode", 1, 10000, 1},
		{"ThreeNodes", 3, 10000, 1},
		{"FiveNodes", 5, 10000, 1},
		{"SingleNode_HighCost", 1, 10000, 10},
		{"ThreeNodes_HighCost", 3, 10000, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runPerformanceTest(t, tc.nodeCount, tc.credits, tc.cost)
		})
	}
}

func runPerformanceTest(t *testing.T, nodeCount int, totalCredits, cost int64) {
	ctx := context.Background()

	h := integration.New(t, integration.Config{
		NumNodes: nodeCount,
	})

	// Set up test resources using seed
	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Remaining:   ptr.P(int32(totalCredits)),
	})

	keyStart := keyResponse.Key

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Key: keyStart,
		Credits: &openapi.KeysVerifyKeyCredits{
			Cost: int32(cost),
		},
	}

	lb := integration.NewLoadbalancer(h)

	// Run simple performance test with fixed number of requests
	numRequests := 100
	successCount := 0
	
	start := time.Now()
	for i := 0; i < numRequests; i++ {
		res, callErr := integration.CallRandomNode[handler.Request, handler.Response](
			lb, "POST", "/v2/keys.verifyKey", headers, req)

		require.NoError(t, callErr, "Request %d failed", i)
		require.Equal(t, 200, res.Status, "Request %d got wrong status", i)
		
		if res.Body.Data.Valid {
			successCount++
		}
	}
	duration := time.Since(start)
	
	rps := float64(numRequests) / duration.Seconds()
	t.Logf("Performance test for %d nodes: %d requests in %v (%.2f RPS), %d successful", 
		nodeCount, numRequests, duration, rps, successCount)
		
	// Verify reasonable performance (at least 10 RPS)
	require.Greater(t, rps, 10.0, "Should achieve at least 10 RPS")
}

// TestUsageLimitLatency measures latency characteristics under different loads
func TestUsageLimitLatency(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	testCases := []struct {
		name        string
		nodeCount   int
		concurrency int
		samples     int
	}{
		{"SingleNode_LowLoad", 1, 1, 100},
		{"SingleNode_MediumLoad", 1, 10, 100},
		{"SingleNode_HighLoad", 1, 50, 100},
		{"MultiNode_LowLoad", 3, 1, 100},
		{"MultiNode_MediumLoad", 3, 10, 100},
		{"MultiNode_HighLoad", 3, 50, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runLatencyTest(t, tc.nodeCount, tc.concurrency, tc.samples)
		})
	}
}

func runLatencyTest(t *testing.T, nodeCount, concurrency, samples int) {
	ctx := context.Background()

	h := integration.New(t, integration.Config{
		NumNodes: nodeCount,
	})

	// Set up test resources using seed
	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	totalCredits := int64(10000) // Enough to not run out during test
	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Remaining:   ptr.P(int32(totalCredits)),
	})

	keyStart := keyResponse.Key

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Key: keyStart,
		Credits: &openapi.KeysVerifyKeyCredits{
			Cost: 1,
		},
	}

	lb := integration.NewLoadbalancer(h)

	// Collect latency measurements
	var mu sync.Mutex
	latencies := make([]time.Duration, 0, samples*concurrency)

	var wg sync.WaitGroup
	requestChan := make(chan struct{}, samples*concurrency)

	// Fill request channel
	for i := 0; i < samples*concurrency; i++ {
		requestChan <- struct{}{}
	}
	close(requestChan)

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range requestChan {
				start := time.Now()

				res, callErr := integration.CallRandomNode[handler.Request, handler.Response](
					lb, "POST", "/v2/keys.verifyKey", headers, req)

				latency := time.Since(start)

				require.NoError(t, callErr)
				require.Equal(t, 200, res.Status)

				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Calculate statistics
	if len(latencies) == 0 {
		t.Fatal("No latency measurements collected")
	}

	var total time.Duration
	min := time.Hour
	max := time.Duration(0)

	for _, lat := range latencies {
		total += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	avg := total / time.Duration(len(latencies))

	// Calculate percentiles
	// Sort latencies
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	p50 := latencies[len(latencies)/2]
	p95Index := int(float64(len(latencies)) * 0.95)
	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	p95 := latencies[p95Index]
	
	p99Index := int(float64(len(latencies)) * 0.99)
	if p99Index >= len(latencies) {
		p99Index = len(latencies) - 1
	}
	p99 := latencies[p99Index]

	t.Logf("Latency statistics for %d nodes, %d concurrency:", nodeCount, concurrency)
	t.Logf("  Samples: %d", len(latencies))
	t.Logf("  Min: %v", min)
	t.Logf("  Avg: %v", avg)
	t.Logf("  P50: %v", p50)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)
	t.Logf("  Max: %v", max)

	// Assert reasonable latency bounds
	require.Less(t, avg, 100*time.Millisecond, "Average latency should be < 100ms")
	require.Less(t, p95, 200*time.Millisecond, "P95 latency should be < 200ms")
	require.Less(t, p99, 500*time.Millisecond, "P99 latency should be < 500ms")

	// Verify all nodes received traffic in multi-node tests
	if nodeCount > 1 {
		lbMetrics := lb.GetMetrics()
		require.Equal(t, nodeCount, len(lbMetrics), "All nodes should have received traffic")
	}
}

// TestUsageLimitThroughput measures maximum throughput under sustained load
func TestUsageLimitThroughput(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	h := integration.New(t, integration.Config{
		NumNodes: 3,
	})

	ctx := context.Background()

	// Set up test resources using seed
	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	totalCredits := int64(100000) // Large number to not run out
	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Remaining:   ptr.P(int32(totalCredits)),
	})

	keyStart := keyResponse.Key

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Key: keyStart,
		Credits: &openapi.KeysVerifyKeyCredits{
			Cost: 1,
		},
	}

	lb := integration.NewLoadbalancer(h)

	// Run sustained load for 10 seconds
	testDuration := 10 * time.Second
	concurrency := 100

	var requestCount int64
	var errorCount int64
	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Since(start) < testDuration {
				res, callErr := integration.CallRandomNode[handler.Request, handler.Response](
					lb, "POST", "/v2/keys.verifyKey", headers, req)

				mu.Lock()
				if callErr != nil || res.Status != 200 {
					errorCount++
				} else {
					requestCount++
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(start)

	rps := float64(requestCount) / actualDuration.Seconds()
	errorRate := float64(errorCount) / float64(requestCount+errorCount) * 100

	t.Logf("Throughput test results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Total requests: %d", requestCount)
	t.Logf("  Errors: %d (%.2f%%)", errorCount, errorRate)
	t.Logf("  Throughput: %.2f RPS", rps)

	// Assert minimum performance thresholds
	require.Greater(t, rps, 100.0, "Should achieve at least 100 RPS")
	require.Less(t, errorRate, 1.0, "Error rate should be < 1%")
}
