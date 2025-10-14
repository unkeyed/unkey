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
	// Set quota to allow only 1 query per window
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueriesPerWindow(1))
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
	t.Skip("Execution time quotas are difficult to test reliably in unit tests due to varying query speeds")
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// Set execution time quota to 1 second per window
	h.SetupAnalytics(workspace.ID, testutil.WithMaxExecutionTimePerWindow(1))
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

	// Wait for data, then run queries repeatedly until execution time quota is exceeded
	time.Sleep(2 * time.Second)

	// Keep running queries until we hit the execution time quota
	var lastStatus int
	for i := 0; i < 100; i++ {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		lastStatus = res.Status
		if res.Status == 429 {
			// Successfully triggered the quota!
			return
		}
		// Small delay between queries
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, 429, lastStatus, "Expected to hit execution time quota after multiple queries")
}
