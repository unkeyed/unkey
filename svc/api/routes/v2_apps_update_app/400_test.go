//nolint:exhaustruct
package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
)

func TestUpdateAppBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.update_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	validID := "app_1234abcd"
	validProject := "payments"
	emptyName := ""
	longName := strings.Repeat("a", 257)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing project and app", req: handler.Request{}},
		{name: "missing app", req: handler.Request{Project: validProject}},
		{name: "missing project", req: handler.Request{App: validID}},
		{name: "app with invalid chars", req: handler.Request{Project: validProject, App: "app.1234"}},
		{name: "app too long", req: handler.Request{Project: validProject, App: strings.Repeat("a", 256)}},
		{name: "project with invalid chars", req: handler.Request{Project: "pay.ments", App: validID}},
		{name: "project too long", req: handler.Request{Project: strings.Repeat("a", 256), App: validID}},
		{name: "slug with uppercase", req: handler.Request{Project: validProject, App: validID, Slug: ptr.P("Payments-Api")}},
		{name: "slug with invalid chars", req: handler.Request{Project: validProject, App: validID, Slug: ptr.P("payments_api")}},
		{name: "slug with leading hyphen", req: handler.Request{Project: validProject, App: validID, Slug: ptr.P("-payments")}},
		{name: "slug with consecutive hyphens", req: handler.Request{Project: validProject, App: validID, Slug: ptr.P("payments--api")}},
		{name: "slug too long", req: handler.Request{Project: validProject, App: validID, Slug: ptr.P(strings.Repeat("a", 257))}},
		{name: "empty name", req: handler.Request{Project: validProject, App: validID, Name: &emptyName}},
		{name: "name too long", req: handler.Request{Project: validProject, App: validID, Name: &longName}},
		{name: "default branch too long", req: handler.Request{Project: validProject, App: validID, DefaultBranch: ptr.P(strings.Repeat("a", 257))}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, "Bad Request", res.Body.Error.Title)
			require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
			require.Greater(t, len(res.Body.Error.Errors), 0)
		})
	}

	t.Run("invalid json", func(t *testing.T) {
		invalidJSON := `{"appId": }`

		req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(invalidJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
	})
}
