package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_create_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestCreateKeyBadRequest(t *testing.T) {

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	// Create API using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing apiId", func(t *testing.T) {
		req := handler.Request{
			// Missing ApiId
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty apiId", func(t *testing.T) {
		req := handler.Request{
			ApiId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("apiId too short", func(t *testing.T) {
		req := handler.Request{
			ApiId: "ab",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("byteLength too small", func(t *testing.T) {
		invalidByteLength := 10 // minimum is 16
		req := handler.Request{
			ApiId:      api.ID,
			ByteLength: &invalidByteLength,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("byteLength too large", func(t *testing.T) {
		invalidByteLength := 300 // maximum is 255
		req := handler.Request{
			ApiId:      api.ID,
			ByteLength: &invalidByteLength,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("prefix too long", func(t *testing.T) {
		invalidPrefix := "this_prefix_is_way_too_long_for_the_api" // max is 16
		req := handler.Request{
			ApiId:  api.ID,
			Prefix: &invalidPrefix,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("negative expires timestamp", func(t *testing.T) {
		invalidExpires := int64(-1)
		req := handler.Request{
			ApiId:   api.ID,
			Expires: &invalidExpires,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty permission in list", func(t *testing.T) {
		emptyPermissions := []string{""}
		req := handler.Request{
			ApiId:       api.ID,
			Permissions: &emptyPermissions,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty role in list", func(t *testing.T) {
		emptyRoles := []string{""}
		req := handler.Request{
			ApiId: api.ID,
			Roles: &emptyRoles,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("permission too long", func(t *testing.T) {
		// Create a permission string that's longer than 512 characters
		longPermission := strings.Repeat("a", 513)
		longPermissions := []string{longPermission}
		req := handler.Request{
			ApiId:       api.ID,
			Permissions: &longPermissions,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("role too long", func(t *testing.T) {
		// Create a role string that's longer than 512 characters
		req := handler.Request{
			ApiId: api.ID,
			Roles: ptr.P([]string{strings.Repeat("a", 513)}),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("credits.remaining null with refill should error", func(t *testing.T) {
		req := handler.Request{
			ApiId: api.ID,
			Credits: &openapi.KeyCreditsData{
				Remaining: nullable.NewNullNullable[int64](),
				Refill: &openapi.KeyCreditsRefill{
					Amount:   100,
					Interval: openapi.KeyCreditsRefillIntervalDaily,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "credits.remaining")
	})
}
