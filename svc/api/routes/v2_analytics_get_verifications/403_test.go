package handler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func Test403_NoAnalyticsPermission(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

	route := &Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications_v1",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}

func Test403_CanonicalPermissionMustGrantLogRead(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	route := &Handler{DB: h.DB, AnalyticsConnectionManager: h.AnalyticsConnectionManager, Caches: h.Caches}
	h.Register(route)

	base := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s", workspace.ID, api.ProjectID, api.KeyAuthID.String)
	for name, permission := range map[string]string{
		"parent keyspace read": base + "#read",
		"wrong log action":     base + "/logs#write",
	} {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, permission)
			res := testutil.CallRoute[Request, Response](h, route, analyticsHeaders(rootKey), Request{
				Query: "SELECT key_space_id FROM key_verifications_v1",
			})
			require.Equal(t, http.StatusForbidden, res.Status, "body: %s", res.RawBody)
		})
	}
}
