package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_set_override"
)

// TestSetOverrideAuthorizesCanonicalWriteRatelimitOverride guarantees a
// project-scoped override wildcard can create an override.
func TestSetOverrideAuthorizesCanonicalWriteRatelimitOverride(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	projectID := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID}).ProjectID
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	namespaceName := uid.New("namespace")
	require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, NamespaceCache: h.Caches.RatelimitNamespace}
	h.Register(route)
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/ratelimits/namespaces/%s/overrides/*#write_ratelimit_override", workspaceID, projectID, namespaceID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Namespace:  namespaceName,
		Identifier: "canonical-identifier",
		Limit:      10,
		Duration:   1_000,
	})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.NotEmpty(t, res.Body.Data.OverrideId)
}
