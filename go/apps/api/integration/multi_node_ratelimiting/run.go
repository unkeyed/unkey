package multi_node_ratelimiting

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// RunRateLimitTest runs a single rate limit test with the specified parameters
func RunRateLimitTest(
	t *testing.T,
	h *integration.Harness,
	limit int64,
	duration int64, // milliseconds
	windowCount int,
	loadFactor float64,
	nodeCount int,
) {
	ctx := context.Background()

	// Step 1: Set up test resources
	// -----------------------------

	// Create a namespace for rate limiting
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	namespaceName := uid.New("test")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create a root key for authentication
	rootKey := h.Seed.CreateRootKey(ctx, h.Seed.Resources.UserWorkspace.ID,
		fmt.Sprintf("ratelimit.%s.limit", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a unique identifier for this test
	identifier := uid.New("test")

	// Step 2: Calculate test parameters
	// --------------------------------

	// Calculate requests per second based on load factor and limit/duration
	rps := calculateRPS(limit, duration, loadFactor)

	// Calculate total test duration in seconds
	testDurationSeconds := int(math.Ceil(float64(windowCount) * float64(duration) / 1000.0))

	// Total number of requests to send
	totalRequests := rps * testDurationSeconds

	// Prepare the rate limit request
	req := handler.Request{
		Namespace:  namespaceName,
		Identifier: identifier,
		Limit:      limit,
		Duration:   duration,
		Cost:       ptr.P[int64](1),
	}

	// Step 3: Calculate expected limits
	// --------------------------------

	// Number of time windows that will occur during the test
	numWindows := float64(testDurationSeconds*1000) / float64(duration)

	// Maximum theoretical allowed requests across all windows
	maxAllowed := math.Min(numWindows*float64(limit), float64(totalRequests))

	// Set acceptance thresholds with 5% tolerance
	upperLimit := int(maxAllowed * 1.1)
	lowerLimit := int(maxAllowed * 0.95)

	// Special case: When request rate is below the limit,
	// almost all requests should succeed
	rpsPerWindow := float64(rps) * (float64(duration) / 1000.0)
	if rpsPerWindow <= float64(limit) {
		lowerLimit = int(float64(totalRequests) * 0.95)
		upperLimit = totalRequests
	}

	// Ensure upper limit doesn't exceed total requests
	upperLimit = min(upperLimit, totalRequests)

	// Step 4: Run the load test
	// ------------------------

	t.Logf("Configuration: limit=%d, duration=%s, load=%.1fx", limit, time.Duration(duration)*time.Millisecond, loadFactor)
	t.Logf("Sending %d requests at %d RPS across %d nodes", totalRequests, rps, nodeCount)

	realStart := time.Now()

	// Use simulated clock to speed up the test
	clk := clock.NewTestClock()
	simulatedStart := clk.Now()

	// Track successful requests
	successCount := 0

	// Calculate interval between requests to achieve desired RPS
	interval := time.Second / time.Duration(rps)

	// Create a load balancer to distribute requests across nodes
	lb := integration.NewLoadbalancer(h)

	// Send requests
	for i := 0; i < totalRequests; i++ {
		// Advance simulated clock
		clk.Tick(interval)

		// Set test time header for consistent timing
		headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))

		// Send request to a random node
		res, err := integration.CallRandomNode[handler.Request, handler.Response](
			lb, "POST", "/v2/ratelimit.limit", headers, req)

		require.NoError(t, err)
		require.Equal(t, 200, res.Status, "expected 200 status")

		if res.Body.Data.Success {
			successCount++
		}
	}

	// Step 5: Verify results
	// --------------------

	simulatedDuration := clk.Now().Sub(simulatedStart)
	realDuration := time.Since(realStart)

	t.Logf("Load test simulated %s in %s (%.2f%%)",
		simulatedDuration, realDuration, float64(simulatedDuration)/float64(realDuration)*100.0)
	t.Logf("Successful requests: %d/%d (%.2f%%)",
		successCount, totalRequests, float64(successCount)/float64(totalRequests)*100.0)

	// Verify all nodes received traffic
	lbMetrics := lb.GetMetrics()
	require.Equal(t, nodeCount, len(lbMetrics), "all nodes should have received traffic")

	// Verify success count is within expected range
	require.GreaterOrEqual(t, successCount, lowerLimit,
		"Success count should be >= lower limit (%d)", lowerLimit)
	require.LessOrEqual(t, successCount, upperLimit,
		"Success count should be <= upper limit (%d)", upperLimit)
}

// calculateRPS determines the requests per second based on the rate limit parameters
func calculateRPS(limit int64, duration int64, loadFactor float64) int {
	rps := int(math.Ceil(float64(limit) * loadFactor * (1000.0 / float64(duration))))

	// Must have at least 1 RPS
	if rps < 1 {
		rps = 1
	}

	return rps
}
