package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetUsageBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "workspace.*.read_usage")

	for name, tc := range map[string]struct{ body, location string }{
		"unknown period": {body: `{"period":"lastWeek"}`, location: "/properties/period/enum"},
		"unknown field":  {body: `{"month":"2026-09"}`, location: "/additionalProperties"},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewBufferString(tc.body))
			req.Header = headers(rootKey)
			res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
			require.Len(t, res.Body.Error.Errors, 1)
			require.Equal(t, tc.location, res.Body.Error.Errors[0].Location)
		})
	}
}
