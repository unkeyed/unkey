package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteIdentityUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Invalid authorization token
	t.Run("invalid auth token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
	})
}
