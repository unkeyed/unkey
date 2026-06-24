//nolint:exhaustruct
package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		CtrlClient: &testutil.MockProjectClient{},
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.create_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing name", req: handler.Request{Slug: "payments-service"}},
		{name: "missing slug", req: handler.Request{Name: "Payments Service"}},
		{name: "slug with uppercase", req: handler.Request{Name: "Payments", Slug: "Payments-Service"}},
		{name: "slug with invalid chars", req: handler.Request{Name: "Payments", Slug: "payments.service"}},
		{name: "slug with leading hyphen", req: handler.Request{Name: "Payments", Slug: "-payments"}},
		{name: "slug with trailing hyphen", req: handler.Request{Name: "Payments", Slug: "payments-"}},
		{name: "slug with leading underscore", req: handler.Request{Name: "Payments", Slug: "_payments"}},
		{name: "slug with consecutive hyphens", req: handler.Request{Name: "Payments", Slug: "pay--ments"}},
		{name: "slug with consecutive underscores", req: handler.Request{Name: "Payments", Slug: "pay__ments"}},
		{name: "slug too long", req: handler.Request{Name: "Payments", Slug: strings.Repeat("a", 257)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.NotNil(t, res.Body)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.Equal(t, "Bad Request", res.Body.Error.Title)
			require.Equal(t, "POST request body for '/v2/projects.createProject' failed to validate schema", res.Body.Error.Detail)
			require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
			require.Greater(t, len(res.Body.Error.Errors), 0)
		})
	}

	t.Run("invalid json", func(t *testing.T) {
		invalidJSON := `{"name": "Payments", "slug": }`

		req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(invalidJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Error.Detail)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
	})

	t.Run("missing authorization", func(t *testing.T) {
		noAuthHeaders := http.Header{
			"Content-Type": {"application/json"},
		}

		req := handler.Request{Name: "Payments Service", Slug: "payments-service"}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, noAuthHeaders, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 when authorization header is missing")
	})
}
