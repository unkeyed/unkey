package logdrains_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_get_logdrain"
)

func TestGetRejectsMalformedJSON(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(`{"logdrainId":`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	result := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
	require.Equal(t, http.StatusBadRequest, result.Status, "%s", result.RawBody)
	require.Equal(t, http.StatusBadRequest, result.Body.Error.Status)
	require.NotEmpty(t, result.Body.Meta.RequestId)
}
