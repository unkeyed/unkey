package multi_node_ratelimiting_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/attack"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAccuracy(t *testing.T) {

	// How many nodes to simulate
	nodes := []int{1, 3, 27}

	limits := []int64{5, 100}

	durations := []time.Duration{
		1 * time.Second,
		5 * time.Second,
	}

	// Define load patterns as multipliers of the limit
	loadFactors := []float64{0.9, 10.0}

	testDurations := []time.Duration{
		time.Minute,
	}

	for _, nodeCount := range nodes {
		for _, testDuration := range testDurations {
			for _, limit := range limits {
				for _, duration := range durations {
					for _, loadFactor := range loadFactors {

						t.Run(fmt.Sprintf("nodes=%d_test=%s_limit=%d_duration=%s_loadFactor=%f", nodeCount, testDuration, limit, duration, loadFactor), func(t *testing.T) {

							testutil.SkipUnlessIntegration(t)

							ctx := context.Background()

							// Setup a cluster with the specified number of nodes
							h := integration.New(t, integration.Config{
								NumNodes: nodeCount,
							})

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

							// Create auth for the test
							rootKey := h.Seed.CreateRootKey(ctx, h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))
							headers := http.Header{
								"Content-Type":  {"application/json"},
								"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
							}

							identifier := uid.New("test")

							req := handler.Request{
								Namespace:  namespaceName,
								Limit:      limit,
								Duration:   duration.Milliseconds(),
								Identifier: identifier,
							}

							// Calculate test parameters
							// RPS based on loadFactor and limit/duration
							rps := int(math.Ceil(float64(limit) * loadFactor * (1000.0 / float64(duration.Milliseconds()))))

							// Total seconds needed to cover windowCount windows

							total := 0
							passed := 0

							lb := integration.NewLoadbalancer(h)

							errors := atomic.Int64{}

							results := attack.Attack[integration.TestResponse[handler.Response]](t, attack.Rate{Freq: rps, Per: time.Second}, testDuration, func() integration.TestResponse[handler.Response] {
								res, err := integration.CallRandomNode[handler.Request, handler.Response](lb, "POST", "/v2/ratelimit.limit", headers, req)

								if err != nil {
									errors.Add(1)
								}
								return res
							})

							require.Less(t, errors.Load(), int64(5))

							for res := range results {
								require.Equal(t, http.StatusOK, res.Status, "expected 200 status, but got:%s", res.RawBody)
								total++
								if res.Body.Success {
									passed++
								}

							}

							windows := math.Ceil(float64(testDuration) / float64(duration))
							// Calculate theoretical maximum allowed requests
							maxAllowed := math.Min(windows*float64(limit), float64(total))

							// Calculate limits with some tolerance
							upperLimit := int(maxAllowed * 1.2)
							lowerLimit := int(math.Min(maxAllowed*0.95, float64(total)))

							t.Logf("total: %d, passed: %d, acceptable: [%d - %d]", total, passed, lowerLimit, upperLimit)
							// Verify results
							require.GreaterOrEqual(t, passed, lowerLimit,
								"Success count should be >= lower limit")
							require.LessOrEqual(t, passed, upperLimit,
								"Success count should be <= upper limit")

							t.Logf("balance: %+v", lb.GetMetrics())

						})
					}
				}
			}
		}
	}

}
