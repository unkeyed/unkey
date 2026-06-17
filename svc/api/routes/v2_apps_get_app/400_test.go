package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_get_app"
)

func TestGetAppValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing projectSlug", req: handler.Request{Slug: "app-slug"}},
		{name: "missing slug", req: handler.Request{ProjectSlug: "payments"}},
		{name: "slug uppercase", req: handler.Request{ProjectSlug: "payments", Slug: "App-Slug"}},
		{name: "slug underscore", req: handler.Request{ProjectSlug: "payments", Slug: "app_slug"}},
		{name: "slug leading hyphen", req: handler.Request{ProjectSlug: "payments", Slug: "-app"}},
		{name: "slug consecutive hyphens", req: handler.Request{ProjectSlug: "payments", Slug: "ap--p"}},
		{name: "projectSlug uppercase", req: handler.Request{ProjectSlug: "Payments", Slug: "app-slug"}},
		{name: "slug too long", req: handler.Request{ProjectSlug: "payments", Slug: strings.Repeat("a", 257)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		})
	}
}
