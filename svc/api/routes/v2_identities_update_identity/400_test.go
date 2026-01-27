package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_update_identity"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
	}

	t.Run("missing externalId", func(t *testing.T) {
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Meta: &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.Equal(t, 400, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty externalId", func(t *testing.T) {
		meta := map[string]interface{}{
			"test": "value",
		}

		req := handler.Request{
			Identity: "",
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.Equal(t, 400, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("duplicate ratelimit names", func(t *testing.T) {
		externalID := "identity_123"
		ratelimits := []openapi.RatelimitRequest{
			{
				Name:      "api_calls",
				Limit:     100,
				Duration:  60000,
				AutoApply: true,
			},
			{
				Name:      "api_calls", // Duplicate name
				Limit:     200,
				Duration:  120000,
				AutoApply: true,
			},
		}

		req := handler.Request{
			Identity:   externalID,
			Ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "api_calls")
	})

	t.Run("metadata too large", func(t *testing.T) {
		externalID := "identity_123"

		// Create a large metadata object (over 1MB)
		largeString := strings.Repeat("a", 1024*1024)
		largeMeta := map[string]interface{}{
			"large_field": largeString,
		}

		req := handler.Request{
			Identity: externalID,
			Meta:     &largeMeta,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Metadata is too large")
	})
}
