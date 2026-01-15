package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
)

func Test422_ExceedsMaxMemory(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Set up analytics with very low MaxQueryMemoryBytes (10KB - very restrictive)
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueryMemoryBytes(10_000))
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := h.Clock.Now().UnixMilli()

	// Buffer many verifications to ensure memory usage exceeds limit
	for i := range 50_000 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
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

	// Use a memory-intensive query with aggregation and grouping
	req := Request{
		Query: "SELECT key_id, region, outcome, COUNT(*) as count FROM key_verifications_v1 GROUP BY key_id, region, outcome",
	}

	// Wait for data to be buffered and flushed to ClickHouse, then verify query fails with 422
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 422, res.Status)
	}, 30*time.Second, 500*time.Millisecond)
}
