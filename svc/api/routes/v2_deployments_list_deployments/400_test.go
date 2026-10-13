package handler_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

func TestListValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))
	rootKey := setup.RootKey

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "app without project", req: handler.Request{App: rid("payments-api")}},
		{name: "environment without app", req: handler.Request{Project: rid("payments"), Environment: rid("production")}},
		{name: "environment without project or app", req: handler.Request{Environment: rid("production")}},
		{name: "branch without app", req: handler.Request{Project: rid(setup.Project.Slug), Branch: &[]string{"main"}}},
		{name: "branch without project or app", req: handler.Request{Branch: &[]string{"main"}}},
		{name: "empty branch name", req: handler.Request{Project: rid("payments"), App: rid("payments-api"), Branch: &[]string{""}}},
		{name: "startTime equal to endTime", req: handler.Request{StartTime: new(int64(1_704_067_200_000)), EndTime: new(int64(1_704_067_200_000))}},
		{name: "startTime after endTime", req: handler.Request{StartTime: new(int64(1_704_672_000_000)), EndTime: new(int64(1_704_067_200_000))}},
		{name: "too many branches", req: handler.Request{Project: rid("payments"), App: rid("payments-api"), Branch: new(slices.Repeat([]string{"main"}, 51))}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(rootKey), tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		})
	}
}
