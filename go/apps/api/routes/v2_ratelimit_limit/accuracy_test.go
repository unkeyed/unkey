package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testflags"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestRateLimitAccuracy(t *testing.T) {
	testflags.SkipUnlessIntegration(t)

	// Define test matrices for each dimension
	limits := []int64{
		5,
		100,
	}

	durations := []int64{
		1 * time.Second.Milliseconds(),
		5 * time.Second.Milliseconds(),
		// 1 *time.Minute.Milliseconds(),
		// 5 * time.Minute.Milliseconds(),
		// 1 *time.Hour.Milliseconds(),
		// 24 * time.Minute.Milliseconds(),
	}

	// Define load patterns as multipliers of the limit
	loadFactors := []float64{
		0.9,  // Slightly below limit
		1.0,  // At limit
		10.0, // Well above limit

	}

	// Number of windows to test (determines test duration)
	windowCounts := []int{
		50,
	}

	for _, windows := range windowCounts {
		t.Run(fmt.Sprintf("windows_%d", windows), func(t *testing.T) {
			for _, limit := range limits {
				t.Run(fmt.Sprintf("limit_%d", limit), func(t *testing.T) {
					for _, duration := range durations {
						t.Run(fmt.Sprintf("duration_%dms", duration), func(t *testing.T) {
							for _, loadFactor := range loadFactors {
								t.Run(fmt.Sprintf("load_%.1fx", loadFactor), func(t *testing.T) {
									t.Parallel()
									h := testutil.NewHarness(t)

									route := handler.New(handler.Services{
										DB:          h.DB,
										Keys:        h.Keys,
										Logger:      h.Logger,
										Permissions: h.Permissions,
										Ratelimit:   h.Ratelimit,
									})
									h.Register(route)
									ctx := context.Background()

									// Create a namespace
									namespaceID := uid.New(uid.RatelimitNamespacePrefix)
									namespaceName := uid.New("test")
									err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
										ID:          namespaceID,
										WorkspaceID: h.Resources.UserWorkspace.ID,
										Name:        namespaceName,
										CreatedAt:   time.Now().UnixMilli(),
									})
									require.NoError(t, err)

									rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

									headers := http.Header{
										"Content-Type":  {"application/json"},
										"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
									}

									// Create a unique identifier for this test
									identifier := uid.New("test")

									// Calculate test parameters
									// RPS based on loadFactor and limit/duration
									rps := int(math.Ceil(float64(limit) * loadFactor * (1000.0 / float64(duration))))

									// Must have at least 1 RPS
									if rps < 1 {
										rps = 1
									}

									// Total seconds needed to cover windowCount windows
									seconds := int(math.Ceil(float64(windows) * float64(duration) / 1000.0))

									// Request that will be sent repeatedly
									req := handler.Request{
										Namespace:  namespaceName,
										Identifier: identifier,
										Limit:      limit,
										Duration:   duration,
									}

									// Calculate number of windows and expected limits
									totalRequests := rps * seconds
									numWindows := float64(seconds*1000) / float64(duration)

									// Calculate theoretical maximum allowed requests
									maxAllowed := numWindows * float64(limit)
									maxAllowed = math.Min(maxAllowed, float64(totalRequests))

									// Calculate limits with some tolerance
									upperLimit := int(maxAllowed * 1.2)
									lowerLimit := int(maxAllowed * 0.95)

									// Special case for below-limit scenarios
									rpsPerWindow := float64(rps) * (float64(duration) / 1000.0)
									if rpsPerWindow <= float64(limit) {
										// When below limit, we expect all or nearly all to succeed
										lowerLimit = int(float64(totalRequests) * 0.95)
										upperLimit = totalRequests
									}

									// Cap at total requests
									upperLimit = min(upperLimit, totalRequests)

									realStart := time.Now()
									// Run load test
									start := h.Clock.Now()
									successCount := 0

									// Calculate interval between requests to achieve desired RPS
									interval := time.Second / time.Duration(rps)

									t.Logf("sending %d requests", totalRequests)
									for i := 0; i < totalRequests; i++ {
										// Simulate request timing to achieve target RPS
										h.Clock.Tick(interval)

										res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
										require.Equal(t, 200, res.Status, "expected 200 status")

										if res.Body.Success {
											successCount++
										}
									}

									simulatedDuration := h.Clock.Now().Sub(start)
									realDuration := time.Since(realStart)

									t.Logf("Load test simulated %s in %s (%.2f%%)",
										simulatedDuration, realDuration, float64(simulatedDuration)/float64(realDuration)*100.0)

									// Verify results
									require.GreaterOrEqual(t, successCount, lowerLimit,
										"Success count should be >= lower limit")
									require.LessOrEqual(t, successCount, upperLimit,
										"Success count should be <= upper limit")
								})
							}
						})
					}
				})
			}
		})
	}
}
