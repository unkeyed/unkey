package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// Test200_URQL_LogicalTable confirms that a URQL query referencing the
// logical `key_verifications` table is compiled, secured, and executed
// end-to-end against ClickHouse.
func Test200_URQL_LogicalTable(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := time.Now().UnixMilli()
	for i := range 5 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
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
		DB:                         h.DB,
		Keys:                       h.Keys,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	time.Sleep(5 * time.Second)

	req := Request{
		Query: "SELECT outcome, count() AS c FROM key_verifications WHERE time > now() - INTERVAL 1 HOUR GROUP BY outcome",
	}
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
	// columnFormats should be absent when no prettyFormat() was used.
	require.Nil(t, res.Body.ColumnFormats)
}

// Test200_URQL_PrettyFormatColumnHints confirms that prettyFormat() calls
// emit column-format hints in meta.
func Test200_URQL_PrettyFormatColumnHints(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := time.Now().UnixMilli()
	for i := range 3 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
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
		DB:                         h.DB,
		Keys:                       h.Keys,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	time.Sleep(5 * time.Second)

	req := Request{
		Query: "SELECT outcome, prettyFormat(count(), 'quantity') AS total FROM key_verifications WHERE time > now() - INTERVAL 1 HOUR GROUP BY outcome",
	}
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.ColumnFormats)
	require.Equal(t, "quantity", (*res.Body.ColumnFormats)["total"])
}

// Test200_URQL_SelfJoin confirms a JOIN URQL query compiles, secures, and
// executes end-to-end. Both join sides resolve to the same physical variant.
func Test200_URQL_SelfJoin(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := time.Now().UnixMilli()
	for i := range 3 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
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
		DB:                         h.DB,
		Keys:                       h.Keys,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	time.Sleep(5 * time.Second)

	req := Request{
		Query: "SELECT a.outcome FROM key_verifications a JOIN key_verifications b ON a.request_id = b.request_id WHERE a.time > now() - INTERVAL 1 HOUR",
	}
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
}

// Test200_URQL_CTE confirms a CTE URQL query compiles, secures, and
// executes end-to-end against ClickHouse. The CTE body gets workspace_id
// injected on the real-table reference; the outer SELECT reading from the
// CTE name does NOT get the filter (which would fail because the CTE
// projection doesn't expose workspace_id).
func Test200_URQL_CTE(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	now := time.Now().UnixMilli()
	for i := range 3 {
		h.KeyVerifications.Buffer(schema.KeyVerification{
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
		DB:                         h.DB,
		Keys:                       h.Keys,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	time.Sleep(5 * time.Second)

	req := Request{
		Query: "WITH recent AS (SELECT outcome FROM key_verifications WHERE time > now() - INTERVAL 1 HOUR) SELECT count() AS c FROM recent",
	}
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
}

// Test200_URQL_FallsBackToLegacy confirms that a query using legacy table
// aliases continues to work unchanged: URQL returns ErrNotURQL and the
// legacy parser handles the query as before.
func Test200_URQL_FallsBackToLegacy(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
		Keys:                       h.Keys,
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
	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 200, res.Status)
	require.Nil(t, res.Body.ColumnFormats)
}
