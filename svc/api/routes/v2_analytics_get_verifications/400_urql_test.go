package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Test400_URQL_UnknownColumn confirms that a URQL query referencing the
// logical `key_verifications` table with an unknown column produces a 400
// (URQL owns the query and surfaces its compile error rather than falling
// through to the legacy parser).
func Test400_URQL_UnknownColumn(t *testing.T) {
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
		Query: "SELECT badcol FROM key_verifications WHERE time > now() - INTERVAL 1 DAY",
	}
	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 400, res.Status, "Unknown column on URQL logical table should return 400")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Detail, "badcol")
}

// Test400_URQL_AllowedValues confirms that a URQL query with a value not in
// the column's AllowedValues list produces a 400 with a friendly error.
func Test400_URQL_AllowedValues(t *testing.T) {
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
		Query: "SELECT outcome FROM key_verifications WHERE outcome = 'NOPE' AND time > now() - INTERVAL 1 DAY",
	}
	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	t.Logf("Status: %d, RawBody: %s", res.Status, res.RawBody)
	require.Equal(t, 400, res.Status, "Invalid allowed value should return 400")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Detail, "NOPE")
}
