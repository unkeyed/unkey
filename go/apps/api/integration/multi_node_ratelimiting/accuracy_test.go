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
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestRateLimitAccuracy(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	nodeCounts := []int{
		1, 3, 9,
	}

	// Define test matrices for each dimension
	limits := []int64{
		5,
		100,
		10000,
	}

	durations := []time.Duration{
		1 * time.Second,
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	// Define load patterns as multipliers of the limit
	loadFactors := []float64{
		0.9,  // Slightly below limit
		1.0,  // At limit
		10.0, // Well above limit

	}

	// Number of windows to test (determines test duration)
	windowCounts := []int{
		100,
	}

	for _, nodes := range nodeCounts {
		t.Run(fmt.Sprintf("nodes_%d", nodes), func(t *testing.T) {
			h := integration.New(t, integration.Config{

				NumNodes: nodes,
			})

			for _, windows := range windowCounts {
				for _, limit := range limits {
					for _, duration := range durations {
						for _, loadFactor := range loadFactors {
							t.Run(fmt.Sprintf("windows=%d_limit=%d_duration=%d_load=%.1fx", windows, limit, duration, loadFactor), func(t *testing.T) {

								ctx := context.Background()

								// Create a namespace
								namespaceID := uid.New(uid.RatelimitNamespacePrefix)
								namespaceName := uid.New("test")
								err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
									ID:          namespaceID,
									WorkspaceID: h.Resources().UserWorkspace.ID,
									Name:        namespaceName,
									CreatedAt:   time.Now().UnixMilli(),
								})
								require.NoError(t, err)

								rootKey := h.Seed.CreateRootKey(ctx, h.Seed.Resources.UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

								headers := http.Header{
									"Content-Type":  {"application/json"},
									"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
								}

								// Create a unique identifier for this test
								identifier := uid.New("test")

								// Calculate test parameters
								// RPS based on loadFactor and limit/duration
								rps := int(math.Ceil(float64(limit) * loadFactor * (1000.0 / float64(duration.Milliseconds()))))

								// Must have at least 1 RPS
								if rps < 1 {
									rps = 1
								}

								// Total seconds needed to cover windowCount windows
								seconds := int(math.Ceil(float64(windows) * float64(duration.Milliseconds()) / 1000.0))

								// Request that will be sent repeatedly
								req := handler.Request{
									Namespace:  namespaceName,
									Identifier: identifier,
									Limit:      limit,
									Duration:   duration.Milliseconds(),
								}

								// Calculate number of windows and expected limits
								totalRequests := rps * seconds
								numWindows := float64(seconds*1000) / float64(duration.Milliseconds())

								// Calculate theoretical maximum allowed requests
								maxAllowed := numWindows * float64(limit)
								maxAllowed = math.Min(maxAllowed, float64(totalRequests))

								// Calculate limits with some tolerance
								upperLimit := int(maxAllowed * 1.05)
								lowerLimit := int(maxAllowed * 0.95)

								// Special case for below-limit scenarios
								rpsPerWindow := float64(rps) * (float64(duration.Milliseconds()) / 1000.0)
								if rpsPerWindow <= float64(limit) {
									// When below limit, we expect all or nearly all to succeed
									lowerLimit = int(float64(totalRequests) * 0.95)
									upperLimit = totalRequests
								}

								// Cap at total requests
								upperLimit = min(upperLimit, totalRequests)

								realStart := time.Now()
								// Run load test
								clk := clock.NewTestClock(realStart)
								simulatedStart := clk.Now()
								successCount := 0

								// Calculate interval between requests to achieve desired RPS
								interval := time.Second / time.Duration(rps)

								lb := integration.NewLoadbalancer(h)

								t.Logf("sending %d requests", totalRequests)
								for i := 0; i < totalRequests; i++ {
									// Simulate request timing to achieve target RPS
									clk.Tick(interval)
									//time.Sleep(interval)

									headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))
									res, err := integration.CallRandomNode[handler.Request, handler.Response](lb, "POST", "/v2/ratelimit.limit", headers, req)
									require.NoError(t, err)
									require.NoError(t, err)
									require.Equal(t, 200, res.Status, "expected 200 status")

									if res.Body.Data.Success {
										successCount++
									}
								}

								simulatedDuration := clk.Now().Sub(simulatedStart)
								realDuration := time.Since(realStart)

								t.Logf("Load test simulated %s in %s (%.2f%%)",
									simulatedDuration, realDuration, float64(simulatedDuration)/float64(realDuration)*100.0)

								lbMetrics := lb.GetMetrics()
								require.Equal(t, nodes, len(lbMetrics), "all nodes should have received traffic")
								// Verify results
								require.GreaterOrEqual(t, successCount, lowerLimit,
									"Success count should be >= lower limit")
								require.LessOrEqual(t, successCount, upperLimit,
									"Success count should be <= upper limit")
							})
						}
					}
				}
			}
		})

	}
}
