package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_get_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestInternalError(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		Vault:        h.Vault,
		LiveKeyCache: h.Caches.LiveKeyByID,
	}
	h.Register(route)
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("database connection closed during request", func(t *testing.T) {
		// Close the database connections to simulate a database failure
		err := h.DB.Close()
		require.NoError(t, err)
		req := handler.Request{
			KeyId:   uid.New(uid.KeyPrefix),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not load the requested key")
	})
}
