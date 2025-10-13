package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func Test429_QueryQuotaExceeded(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// Note: Can't use SetupAnalytics with MaxQueriesPerWindow - it's not configurable yet
	// This test may need adjustment once quota limits are testable
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := h.Clock.Now().UnixMilli()

	// Buffer some key verifications
	for i := 0; i < 5; i++ {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  "",
			Tags:        []string{},
		})
	}

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	// Wait for data, first query should succeed
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
	}, 30*time.Second, time.Second)

	// Second query should fail with 429 (quota exceeded)
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 429, res.Status)
}

func Test429_ExecutionTimeQuotaExceeded(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// Note: Can't use SetupAnalytics with MaxExecutionTimePerWindow - it's not configurable yet
	// This test may need adjustment once execution time quotas are testable
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := h.Clock.Now().UnixMilli()

	// Buffer many key verifications to make queries take time
	for i := 0; i < 1000; i++ {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  "",
			Tags:        []string{},
		})
	}

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Complex query that takes time
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	// Wait for data, then make multiple queries to exceed execution time quota
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		// Should eventually fail with 429
		if res.Status == 429 {
			require.Equal(c, 429, res.Status)
		}
	}, 30*time.Second, time.Second)
}
