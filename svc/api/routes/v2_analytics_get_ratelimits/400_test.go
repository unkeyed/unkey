package handler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
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

// Test400_InvalidQueriesAndNamespaceRequirements guarantees malformed SQL,
// missing positive namespace filters, excess IDs, and physical tables fail.
func Test400_InvalidQueriesAndNamespaceRequirements(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	id := createNamespace(t, h, workspaceID)
	eleven := make([]string, 11)
	for i := range eleven {
		eleven[i] = fmt.Sprintf("'%s'", uid.New(uid.RatelimitNamespacePrefix))
	}
	tests := []string{
		"", "SELECT FROM", "SELECT * FROM ratelimits_v1", fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id != '%s'", id),
		"SELECT * FROM ratelimits_v1 WHERE namespace_id IN (" + strings.Join(eleven, ",") + ")",
		fmt.Sprintf("SELECT * FROM default.ratelimits_raw_v2 WHERE namespace_id = '%s'", id),
	}
	for _, query := range tests {
		res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, auth(rootKey), Request{Query: query})
		require.Equal(t, 400, res.Status, query)
	}
}

// TestAuthorizeAllowsExactlyTenUniqueNamespaces locks the inclusive upper
// bound for namespace authorization.
func TestAuthorizeAllowsExactlyTenUniqueNamespaces(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	ids := make([]string, 10)
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
