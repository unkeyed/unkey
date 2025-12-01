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

func Test200_Success(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := time.Now().UnixMilli()

	// Buffer some key verifications
	for i := range 5 {
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

	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	// Wait for buffered data to be available
	time.Sleep(5 * time.Second)

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
	require.Len(t, res.Body.Data, 1)
}

func Test200_PermissionFiltersByApiId(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	// Create root key with permission ONLY for api1
	rootKey := h.CreateRootKey(workspace.ID, "api."+api1.ID+".read_analytics")

	now := time.Now().UnixMilli()

	// Buffer verifications for api1
	for i := range 3 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api1.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  "",
			Tags:        []string{},
		})
	}

	// Buffer verifications for api2 (should NOT be returned)
	for i := range 5 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api2.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-east-1",
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

	// Query all verifications - should only return api1's due to permission filter
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	// Wait for buffered data to be available
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)
		require.Len(c, res.Body.Data, 1)

		// Verify the count is 3 (only api1's verifications), not 8 (api1 + api2)
		count, ok := res.Body.Data[0]["count"]
		require.True(c, ok, "count field should exist")
		require.Equal(c, float64(3), count, "should only return verifications for api1")
	}, 30*time.Second, time.Second)
}

func Test200_PermissionFiltersByKeySpaceId(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	// Create root key with permission ONLY for api1
	rootKey := h.CreateRootKey(workspace.ID, "api."+api1.ID+".read_analytics")

	now := time.Now().UnixMilli()

	// Buffer verifications for api1
	for i := range 3 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api1.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  "",
			Tags:        []string{},
		})
	}

	// Buffer verifications for api2 (should NOT be returned)
	for i := range 5 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api2.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-east-1",
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

	// Query with both key_space_ids in WHERE clause
	// Should only return data for api1 due to permission filter
	req := Request{
		Query: "SELECT key_space_id, COUNT(*) as count FROM key_verifications_v1 GROUP BY key_space_id",
	}

	// Wait for buffered data to be available
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)

		// Should only return 1 group (api1's key_space_id), not 2
		require.Len(c, res.Body.Data, 1)

		// Verify it's api1's key_space_id
		keySpaceID, ok := res.Body.Data[0]["key_space_id"]
		require.True(c, ok, "key_space_id field should exist")
		require.Equal(c, api1.KeyAuthID.String, keySpaceID)

		// Verify the count is 3
		count, ok := res.Body.Data[0]["count"]
		require.True(c, ok, "count field should exist")
		require.Equal(c, float64(3), count)
	}, 30*time.Second, time.Second)
}
func Test200_QueryWithin30DaysRetention(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := time.Now().UnixMilli()

	// Buffer verification from 7 days ago (within 30-day retention)
	h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
		RequestID:   uid.New(uid.RequestPrefix),
		Time:        now - (7 * 24 * 60 * 60 * 1000), // 7 days ago
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		KeyID:       uid.New(uid.KeyPrefix),
		Region:      "us-west-1",
		Outcome:     "VALID",
		IdentityID:  "",
		Tags:        []string{},
	})

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

	// Query last 7 days (within 30-day retention)
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1 WHERE time >= now() - INTERVAL 7 DAY",
	}

	time.Sleep(5 * time.Second) // Wait for data

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Query within retention should succeed")
	require.NotNil(t, res.Body)
}

func Test200_QueryAtExact30DayRetentionLimit(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

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

	// Query exactly 30 days (at retention limit)
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1 WHERE time >= now() - INTERVAL 30 DAY",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Query at retention limit should succeed")
	require.NotNil(t, res.Body)
}

func Test200_QueryWithCustomRetention90Days(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90)) // 90-day retention
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

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

	// Query 60 days (within 90-day retention)
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1 WHERE time >= now() - INTERVAL 60 DAY",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Query within custom retention should succeed")
	require.NotNil(t, res.Body)
}

func Test200_RLSWorkspaceIsolation(t *testing.T) {
	h := testutil.NewHarness(t)

	// Create two separate workspaces
	workspace1 := h.CreateWorkspace()
	workspace2 := h.CreateWorkspace()

	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace1.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace2.ID,
	})

	// Setup analytics for both workspaces
	h.SetupAnalytics(workspace1.ID)
	h.SetupAnalytics(workspace2.ID)

	rootKey1 := h.CreateRootKey(workspace1.ID, "api.*.read_analytics")

	// Use actual current time for analytics data since ClickHouse's now() uses real time, not mock clock
	now := time.Now().UnixMilli()

	// Buffer data for workspace 1
	for i := range 5 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace1.ID,
			KeySpaceID:  api1.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  "",
			Tags:        []string{},
		})
	}

	// Buffer data for workspace 2 (should NOT be accessible by workspace1's key)
	for i := range 10 {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace2.ID,
			KeySpaceID:  api2.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-east-1",
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
		"Authorization": []string{"Bearer " + rootKey1},
		"Content-Type":  []string{"application/json"},
	}

	// Query all verifications - should only return workspace1's data due to RLS
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1 WHERE time >= now() - INTERVAL 1 DAY",
	}

	time.Sleep(10 * time.Second) // Wait for data to be flushed to ClickHouse

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
	require.Len(t, res.Body.Data, 1)

	// Verify only workspace1's data is returned (5 verifications), not workspace2's (10)
	count, ok := res.Body.Data[0]["count"]
	require.True(t, ok)
	require.Equal(t, float64(5), count, "RLS should filter to only workspace1's data")
}

func Test200_QueryWithoutTimeFilter_AutoAddsFilter(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

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

	// Query without time filter - should auto-add time >= now() - INTERVAL 30 DAY
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	if res.Status != 200 {
		t.Logf("Response status: %d", res.Status)
		t.Logf("Response body: %s", res.RawBody)
	}
	require.Equal(t, 200, res.Status, "Query without time filter should succeed with auto-added filter")
	require.NotNil(t, res.Body)
}
