package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetLimitsWithoutLimitsRow(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:limits#read", workspace.ID))
	_, err := h.DB.RW().ExecContext(t.Context(), "DELETE FROM limits WHERE workspace_id = ?", workspace.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header = bearer(rootKey)
	res := testutil.CallRaw[openapi.InternalServerErrorResponse](h, req)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/service_unavailable", res.Body.Error.Type)
	require.Equal(t, "Resource limits are not configured for this workspace. Contact support@unkey.com.", res.Body.Error.Detail)
}
