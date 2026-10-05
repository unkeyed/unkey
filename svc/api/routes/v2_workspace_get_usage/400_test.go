package handler_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetUsageBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:usage#read", workspace.ID))

	call := func(t *testing.T, body string) testutil.TestResponse[openapi.BadRequestErrorResponse] {
		t.Helper()
		req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewBufferString(body))
		req.Header = headers(rootKey)
		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		return res
	}

	for name, tc := range map[string]struct{ body, location string }{
		"month thirteen":  {body: `{"period":{"year":2026,"month":13}}`, location: "/properties/period/$ref/properties/month/maximum"},
		"month zero":      {body: `{"period":{"year":2026,"month":0}}`, location: "/properties/period/$ref/properties/month/minimum"},
		"missing year":    {body: `{"period":{"month":9}}`, location: "/properties/period/$ref/required"},
		"month as string": {body: `{"period":{"year":2026,"month":"09"}}`, location: "/properties/period/$ref/properties/month/type"},
		"unknown field":   {body: `{"month":"2026-09"}`, location: "/additionalProperties"},
	} {
		t.Run(name, func(t *testing.T) {
			res := call(t, tc.body)
			require.Len(t, res.Body.Error.Errors, 1)
			require.Equal(t, tc.location, res.Body.Error.Errors[0].Location)
		})
	}

	// 90 days before 2026-10-05 is 2026-07-07, so August is the earliest full month
	h.Clock.Set(time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC))

	t.Run("future month", func(t *testing.T) {
		res := call(t, `{"period":{"year":2026,"month":11}}`)
		require.Equal(t, "'period' 2026-11 is in the future. The latest month is 2026-10.", res.Body.Error.Detail)
	})

	t.Run("earliest month is allowed", func(t *testing.T) {
		req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewBufferString(`{"period":{"year":2026,"month":8}}`))
		req.Header = headers(rootKey)
		res := testutil.CallRaw[openapi.V2WorkspaceGetUsageResponseBody](h, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	})

	t.Run("month before compute usage retention", func(t *testing.T) {
		res := call(t, `{"period":{"year":2026,"month":7}}`)
		require.Equal(t, "'period' 2026-07 starts more than 90 days ago. Compute usage is kept for 90 days, so the earliest month is 2026-08.", res.Body.Error.Detail)
	})
}
