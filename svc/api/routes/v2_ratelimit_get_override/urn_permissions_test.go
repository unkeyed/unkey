package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_get_override"
)

// TestGetOverride_AuthorizesURNPermissions guarantees the handler queries the
// exact override resource while accepted URN grants may be exact or broader.
func TestGetOverride_AuthorizesURNPermissions(t *testing.T) {
	ctx := context.Background()
	setup := setupGetOverrideURNPermissionTest(t, ctx)

	tests := []struct {
		name       string
		permission string
	}{
		{
			name: "exact override permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/namespaces/%s/overrides/%s#read_override",
				setup.workspaceID,
				setup.namespaceID,
				setup.overrideID,
			),
		},
		{
			name: "namespace override wildcard permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/namespaces/%s/overrides/*#read_override",
				setup.workspaceID,
				setup.namespaceID,
			),
		},
		{
			name: "namespace descendant wildcard permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/namespaces/%s/**#read_override",
				setup.workspaceID,
				setup.namespaceID,
			),
		},
		{
			name: "ratelimits descendant wildcard permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/**#read_override",
				setup.workspaceID,
			),
		},
		{
			name: "workos wildcard namespace permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/namespaces/*/overrides/*#read_override",
				setup.workspaceID,
			),
		},
		{
			name: "admin permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:**#*",
				setup.workspaceID,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootKey := setup.h.CreateRootKey(setup.workspaceID, tt.permission)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](setup.h, setup.route, headers, setup.req)
			require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, setup.overrideID, res.Body.Data.OverrideId)
		})
	}
}

// TestGetOverride_RejectsNonCoveringURNPermissions guarantees URN grants are
// scoped by workspace, resource path, and action before the handler responds,
// and that the rejection is the same 404 a missing override produces so the
// response does not reveal that the override exists.
func TestGetOverride_RejectsNonCoveringURNPermissions(t *testing.T) {
	ctx := context.Background()
	setup := setupGetOverrideURNPermissionTest(t, ctx)

	tests := []struct {
		name       string
		permission string
	}{
		{
			name: "sibling namespace wildcard permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/namespaces/ns_other/overrides/*#read_override",
				setup.workspaceID,
			),
		},
		{
			name: "wrong action permission",
			permission: fmt.Sprintf(
				"unkey:v1:%s:ratelimits/namespaces/%s/overrides/%s#delete_override",
				setup.workspaceID,
				setup.namespaceID,
				setup.overrideID,
			),
		},
		{
			name: "wrong workspace permission",
			permission: fmt.Sprintf(
				"unkey:v1:ws_other:ratelimits/namespaces/%s/overrides/%s#read_override",
				setup.namespaceID,
				setup.overrideID,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootKey := setup.h.CreateRootKey(setup.workspaceID, tt.permission)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](setup.h, setup.route, headers, setup.req)
			require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_override_not_found", res.Body.Error.Type)
		})
	}
}

type getOverrideURNPermissionTest struct {
	h           *testutil.Harness
	route       *handler.Handler
	req         handler.Request
	workspaceID string
	namespaceID string
	overrideID  string
}

func setupGetOverrideURNPermissionTest(t *testing.T, ctx context.Context) getOverrideURNPermissionTest {
	t.Helper()

	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID

	namespaceID := uid.New("test_ns")
	namespaceName := uid.New("test")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: workspaceID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	identifier := "test_identifier"
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: workspaceID,
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       10,
		Duration:    1000,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	return getOverrideURNPermissionTest{
		h:           h,
		route:       route,
		req:         handler.Request{Namespace: namespaceName, Identifier: identifier},
		workspaceID: workspaceID,
		namespaceID: namespaceID,
		overrideID:  overrideID,
	}
}
