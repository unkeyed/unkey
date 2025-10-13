package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func Test422_ExceedsMaxResultRows(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// Set up analytics with very low MaxQueryResultRows
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueryResultRows(1))
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := h.Clock.Now().UnixMilli()

	// Buffer multiple key verifications to exceed the limit
	for i := 0; i < 10; i++ {
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
		Query: "SELECT * FROM key_verifications_v1",
	}

	// Wait for data to be buffered
	time.Sleep(2 * time.Second)

	// Query should fail immediately with 422 due to max_result_rows=1 limit
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 422, res.Status)
}

func Test422_ExceedsMaxMemory(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// Set up analytics with very low MaxQueryMemoryBytes
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueryMemoryBytes(1))
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := h.Clock.Now().UnixMilli()

	// Buffer some verifications
	for i := 0; i < 100; i++ {
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
		Query: "SELECT * FROM key_verifications_v1",
	}

	// Wait for data to be buffered
	time.Sleep(2 * time.Second)

	// Query should fail immediately with 422 due to max_memory_usage=1 limit
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 422, res.Status)
}

func Test422_ExceedsMaxRowsToRead(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// Set up analytics with very low MaxRowsToRead
	h.SetupAnalytics(workspace.ID, testutil.WithMaxRowsToRead(1))
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := h.Clock.Now().UnixMilli()

	// Buffer multiple verifications
	for i := 0; i < 10; i++ {
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
		Query: "SELECT * FROM key_verifications_v1",
	}

	// Wait for data to be buffered
	time.Sleep(2 * time.Second)

	// Query should fail immediately with 422 due to max_rows_to_read=1 limit
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 422, res.Status)
}
