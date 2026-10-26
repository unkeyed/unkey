package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

func TestDeletePermissionRequiresNonemptyPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "rbac.*.delete_permission")

	type request struct {
		Permission *string `json:"permission,omitempty"`
	}
	for _, tc := range []struct {
		name string
		body request
	}{
		{name: "missing", body: request{}},
		{name: "empty", body: request{Permission: new("")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[request, openapi.BadRequestErrorResponse](
				h, route, testutil.Bearer(rootKey), tc.body,
			)
			require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.Contains(t, res.Body.Error.Detail, "validate schema")
			require.NotEmpty(t, res.Body.Error.Errors)
			require.Empty(t, h.FindAuditLogs(t.Context(), t))
		})
	}
}
