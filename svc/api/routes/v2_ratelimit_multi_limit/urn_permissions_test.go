package v2_ratelimit_multi_limit_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
)

// TestMultiLimitAuthorizesCanonicalRatelimitNamespacePermissions guarantees
// canonical write and limit actions cover namespace creation in a batch.
func TestMultiLimitAuthorizesCanonicalRatelimitNamespacePermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	projectID := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID}).ProjectID
	route := &handler.Handler{
		DB:              h.DB,
		RatelimitEvents: h.RatelimitEvents,
		Ratelimit:       h.Ratelimit,
		NamespaceCache:  h.Caches.RatelimitNamespace,
		Auditlogs:       h.Auditlogs,
	}
	h.Register(route)

	namespaceName := uid.New("namespace")
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/ratelimits/namespaces/*#write_ratelimit_namespace", workspaceID, projectID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/ratelimits/namespaces/*#limit_ratelimit_namespace", workspaceID, projectID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		{
			Namespace:  namespaceName,
			Identifier: "canonical-identifier",
			Limit:      10,
			Duration:   1_000,
		},
	})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.True(t, res.Body.Data.Passed)
	namespace, err := db.Query.FindRatelimitNamespaceByName(t.Context(), h.DB.RO(), db.FindRatelimitNamespaceByNameParams{
		WorkspaceID: workspaceID,
		Name:        namespaceName,
	})
	require.NoError(t, err)
	require.Equal(t, projectID, namespace.ProjectID)
}
