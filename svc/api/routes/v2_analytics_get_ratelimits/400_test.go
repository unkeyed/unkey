package handler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Test400_NoAuthHeaderFailsValidation guarantees request validation runs before
// authentication and retains the API's established status precedence.
func Test400_NoAuthHeaderFailsValidation(t *testing.T) {
	h, route, _ := newRoute(t, true)
	query := Request{Query: "SELECT * FROM ratelimits_v1 WHERE namespace_id = 'rlns_missing'"}
	headers := http.Header{"Content-Type": {"application/json"}}

	res := testutil.CallRoute[Request, Response](h, route, headers, query)
	require.Equal(t, 400, res.Status)
}

// Test400_InvalidQueries guarantees malformed SQL, unsupported namespace
// predicates, and physical tables fail.
func Test400_InvalidQueries(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	id := createNamespace(t, h, workspaceID)
	tests := []string{
		"", "SELECT FROM", fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id != '%s'", id),
		fmt.Sprintf("SELECT * FROM default.ratelimits_raw_v2 WHERE namespace_id = '%s'", id),
	}
	for _, query := range tests {
		res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, auth(rootKey), Request{Query: query})
		require.Equal(t, 400, res.Status, query)
	}
}

// TestAuthorizeAllowsManyUniqueNamespaces guarantees namespace authorization no
// longer has a route-specific upper bound.
func TestAuthorizeAllowsManyUniqueNamespaces(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	ids := make([]string, 11)
	for i := range ids {
		ids[i] = createNamespace(t, h, workspaceID)
	}
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + id + "'"
	}
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT count(*) FROM ratelimits_v1 WHERE namespace_id IN (" + strings.Join(quoted, ",") + ")"})
	require.Equal(t, 200, res.Status)
}
