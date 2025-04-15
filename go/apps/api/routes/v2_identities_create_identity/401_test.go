package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestUnauthorizedAccess(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{ExternalId: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{ExternalId: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.NotNil(t, res.Body)
	})

}
