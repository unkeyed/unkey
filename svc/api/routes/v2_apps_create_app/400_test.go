package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_app"
)

func TestCreateAppValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		CtrlClient: &testutil.MockAppClient{},
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	validProject := "payments"

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing project", req: handler.Request{Name: "App", Slug: "app-slug"}},
		{name: "project invalid chars", req: handler.Request{Project: "pay.ments", Name: "App", Slug: "app-slug"}},
		{name: "project too long", req: handler.Request{Project: strings.Repeat("a", 256), Name: "App", Slug: "app-slug"}},
		{name: "missing name", req: handler.Request{Project: validProject, Slug: "app-slug"}},
		{name: "missing slug", req: handler.Request{Project: validProject, Name: "App"}},
		{name: "slug too short", req: handler.Request{Project: validProject, Name: "App", Slug: "ab"}},
		{name: "slug invalid chars", req: handler.Request{Project: validProject, Name: "App", Slug: "app slug"}},
		{name: "slug too long", req: handler.Request{Project: validProject, Name: "App", Slug: strings.Repeat("a", 256)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		})
	}
}
