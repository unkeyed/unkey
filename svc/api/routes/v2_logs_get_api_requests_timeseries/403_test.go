package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_requests_timeseries"
)

// TestGetApiRequestsTimeseriesPermissions guarantees which grants read the request
// logs. The logs belong to the workspace, so a 403 reveals nothing that a 404
// would hide
func TestGetApiRequestsTimeseriesPermissions(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	otherWorkspace := h.CreateWorkspace()
	prefix := "unkey:v1:" + workspaceID + ":"

	tests := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "request logs read", permissions: []string{prefix + "logs/requests#read"}, shouldPass: true},
		{name: "logs container", permissions: []string{prefix + "logs/**#read"}, shouldPass: false},
		{name: "workspace read", permissions: []string{prefix + "**#read"}, shouldPass: true},
		{name: "alongside unrelated grants", permissions: []string{prefix + "limits#read", prefix + "logs/requests#read"}, shouldPass: true},
		{name: "workspace limits", permissions: []string{prefix + "limits#read"}, shouldPass: false},
		{name: "ratelimit logs", permissions: []string{prefix + "projects/*/ratelimits/namespaces/*/logs#read"}, shouldPass: false},
		{name: "other workspace", permissions: []string{"unkey:v1:" + otherWorkspace.ID + ":logs/requests#read"}, shouldPass: false},
		{name: "legacy api permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, tt.permissions...)
			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, auth(rootKey), handler.Request{})

			if tt.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}
