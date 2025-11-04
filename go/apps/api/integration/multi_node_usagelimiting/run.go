package multi_node_usagelimiting

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// RunUsageLimitTest runs a single usage limit test with the specified parameters
func RunUsageLimitTest(
	t *testing.T,
	h *integration.Harness,
	totalCredits int64,
	costPerRequest int64,
	loadFactor float64,
	nodeCount int,
	testDurationSeconds int,
) {
	ctx := context.Background()

	// Step 1: Set up test resources
	// -----------------------------

	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	// Create API using seed
	// nolint: exhaustruct
	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create key with specified credit limit using seed
	// nolint: exhaustruct
	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		//nolint: gosec
		Remaining: ptr.P(int32(totalCredits)),
	})

	keyStart := keyResponse.Key

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Step 2: Calculate test parameters
	// --------------------------------

	// Expected number of successful requests (accounting for cost)
	expectedSuccessful := int(totalCredits / costPerRequest)

	// Calculate requests per second based on load factor
	requestsPerSecond := calculateUsageRPS(totalCredits, costPerRequest, testDurationSeconds, loadFactor)

	// Total number of requests to send
	totalRequests := requestsPerSecond * testDurationSeconds

	// Prepare the key verification request
	req := handler.Request{
		Ratelimits:  nil,
		Permissions: nil,
		Tags:        nil,
		Key:         keyStart,
		Credits: &openapi.KeysVerifyKeyCredits{
			//nolint: gosec
			Cost: int32(costPerRequest),
		},
	}

	// Step 3: Calculate expected limits
	// --------------------------------

	// Set acceptance thresholds
	upperLimit := int(math.Min(float64(expectedSuccessful)*1.05, float64(totalRequests))) // 5% tolerance
	lowerLimit := int(float64(expectedSuccessful) * 0.95)                                 // 5% tolerance for under-counting

	// When load is low, we expect nearly all requests to succeed until credits run out
	if float64(totalRequests) <= float64(expectedSuccessful)*1.1 {
		lowerLimit = min(int(float64(totalRequests)*0.95), expectedSuccessful)
		upperLimit = min(totalRequests, expectedSuccessful)
	}

	// Step 4: Run the load test
	// ------------------------

	t.Logf("Configuration: credits=%d, cost=%d, expectedSuccessful=%d, load=%.1fx",
		totalCredits, costPerRequest, expectedSuccessful, loadFactor)
	t.Logf("Sending %d requests at %d RPS across %d nodes", totalRequests, requestsPerSecond, nodeCount)

	realStart := time.Now()

	// Track successful requests and remaining credits
	successCount := 0
	lastRemaining := totalCredits

	// Create a load balancer to distribute requests across nodes
	lb := integration.NewLoadbalancer(h)

	// Send requests
	for range totalRequests {
		// Send request to a random node
		res, callErr := integration.CallRandomNode[handler.Request, handler.Response](
			lb, "POST", "/v2/keys.verifyKey", headers, req)

		require.NoError(t, callErr)
		require.Equal(t, 200, res.Status, "expected 200 status")

		if res.Body.Data.Valid {
			successCount++
			if res.Body.Data.Credits != nil {
				lastRemaining = int64(*res.Body.Data.Credits)
			}
		} else {
			require.NoError(t, callErr)
			require.Equal(t, openapi.USAGEEXCEEDED, res.Body.Data.Code, "expected USAGEEXCEEDED")
		}

		// No early termination - send all planned requests to verify proper rejection of overconsumption
	}

	// Step 5: Verify results
	// --------------------

	realDuration := time.Since(realStart)

	t.Logf("Load test completed in %s", realDuration)

	// Calculate accuracy
	accuracyPercent := float64(successCount) / float64(expectedSuccessful) * 100.0

	t.Logf("Usage limiting accuracy: %d/%d credits used (%.2f%%), %d/%d requests succeeded",
		successCount, expectedSuccessful, accuracyPercent, successCount, totalRequests)
	t.Logf("Remaining credits: %d (expected: %d)", lastRemaining, totalCredits-int64(successCount)*costPerRequest)

	// Verify all nodes received traffic
	lbMetrics := lb.GetMetrics()
	require.Equal(t, nodeCount, len(lbMetrics), "all nodes should have received traffic")

	// Verify success count is within expected range
	require.GreaterOrEqual(t, successCount, lowerLimit,
		"Success count should be >= lower limit (%d)", lowerLimit)
	require.LessOrEqual(t, successCount, upperLimit,
		"Success count should be <= upper limit (%d)", upperLimit)

	// Verify credit accuracy - remaining credits should be approximately correct
	expectedRemaining := totalCredits - int64(successCount)*costPerRequest
	require.InDelta(t, expectedRemaining, lastRemaining, float64(costPerRequest)*2, // Allow 2 cost units of error
		"Remaining credits should be accurate within 2 cost units")

	// Step 6: Verify ClickHouse Verification data
	// ------------------------------------------

	type aggregatedCounts struct {
		TotalRequests uint64 `ch:"total_requests"`
		SuccessCount  uint64 `ch:"success_count"`
		FailureCount  uint64 `ch:"failure_count"`
	}

	// Check clickhouse
	var chStats aggregatedCounts
	require.Eventually(t, func() bool {
		data, selectErr := clickhouse.Select[aggregatedCounts](
			ctx,
			h.CH.Conn(),
			`SELECT count(*) as total_requests, countIf(outcome = 'VALID') as success_count, countIf(outcome = 'USAGE_EXCEEDED') as failure_count FROM verifications.raw_key_verifications_v1 WHERE workspace_id = {workspace_id:String} AND key_id = {key_id:String}`,
			map[string]string{
				"workspace_id": h.Resources().UserWorkspace.ID,
				"key_id":       keyResponse.KeyID,
			},
		)
		require.NoError(t, selectErr)
		if len(data) != 1 {
			return false
		}
		chStats = data[0]
		//nolint: gosec
		return int(chStats.TotalRequests) == totalRequests
	}, 15*time.Second, 100*time.Millisecond)

	//nolint: gosec
	require.Equal(t, totalRequests, int(chStats.SuccessCount+chStats.FailureCount))
	//nolint: gosec
	require.Equal(t, totalRequests, int(chStats.TotalRequests))
	//nolint: gosec
	require.Equal(t, successCount, int(chStats.SuccessCount))

	// Step 7: Verify Clickhouse Metrics Data
	// ---------------------------------------
	require.Eventually(t, func() bool {
		metricsCount := uint64(0)
		uniqueCount := uint64(0)
		row := h.CH.Conn().QueryRow(ctx, fmt.Sprintf(`SELECT count(*) as total_requests, count(DISTINCT request_id) as unique_requests FROM metrics.raw_api_requests_v1 WHERE workspace_id = '%s';`, h.Resources().UserWorkspace.ID))

		err := row.Scan(&metricsCount, &uniqueCount)
		require.NoError(t, err)

		//nolint: gosec
		return metricsCount == uint64(totalRequests) && uniqueCount == uint64(totalRequests)
	}, 15*time.Second, 100*time.Millisecond)
}

// calculateUsageRPS determines the requests per second for usage limiting tests
func calculateUsageRPS(totalCredits, costPerRequest int64, testDurationSeconds int, loadFactor float64) int {
	// Base RPS: spread the expected successful requests over test duration
	expectedSuccessful := totalCredits / costPerRequest
	baseRPS := int(math.Ceil(float64(expectedSuccessful) / float64(testDurationSeconds)))

	// Apply load factor
	rps := int(math.Ceil(float64(baseRPS) * loadFactor))

	// Must have at least 1 RPS
	if rps < 1 {
		rps = 1
	}

	return rps
}
