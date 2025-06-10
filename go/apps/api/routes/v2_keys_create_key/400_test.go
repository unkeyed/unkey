package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_create_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func Test_CreateKey_BadRequest(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Create API for valid tests
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
	})
	require.NoError(t, err)

	apiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

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
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("empty apiId", func(t *testing.T) {
		req := handler.Request{
			ApiId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("invalid apiId format", func(t *testing.T) {
		req := handler.Request{
			ApiId: "invalid-api-id",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("apiId too short", func(t *testing.T) {
		req := handler.Request{
			ApiId: "ab",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("byteLength too small", func(t *testing.T) {
		invalidByteLength := 10 // minimum is 16
		req := handler.Request{
			ApiId:      apiID,
			ByteLength: &invalidByteLength,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("byteLength too large", func(t *testing.T) {
		invalidByteLength := 300 // maximum is 255
		req := handler.Request{
			ApiId:      apiID,
			ByteLength: &invalidByteLength,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("prefix too long", func(t *testing.T) {
		invalidPrefix := "this_prefix_is_way_too_long_for_the_api" // max is 16
		req := handler.Request{
			ApiId:  apiID,
			Prefix: &invalidPrefix,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("negative expires timestamp", func(t *testing.T) {
		invalidExpires := int64(-1)
		req := handler.Request{
			ApiId:   apiID,
			Expires: &invalidExpires,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("nonexistent permission", func(t *testing.T) {
		nonexistentPermissions := []string{"nonexistent.permission"}
		req := handler.Request{
			ApiId:       apiID,
			Permissions: &nonexistentPermissions,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Permission 'nonexistent.permission' was not found")
	})

	t.Run("nonexistent role", func(t *testing.T) {
		nonexistentRoles := []string{"nonexistent_role"}
		req := handler.Request{
			ApiId: apiID,
			Roles: &nonexistentRoles,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Role 'nonexistent_role' was not found")
	})

	t.Run("empty permission in list", func(t *testing.T) {
		emptyPermissions := []string{""}
		req := handler.Request{
			ApiId:       apiID,
			Permissions: &emptyPermissions,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("empty role in list", func(t *testing.T) {
		emptyRoles := []string{""}
		req := handler.Request{
			ApiId: apiID,
			Roles: &emptyRoles,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("permission too long", func(t *testing.T) {
		// Create a permission string that's longer than 512 characters
		longPermission := strings.Repeat("a", 513)
		longPermissions := []string{longPermission}
		req := handler.Request{
			ApiId:       apiID,
			Permissions: &longPermissions,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})

	t.Run("role too long", func(t *testing.T) {
		// Create a role string that's longer than 512 characters
		longRole := strings.Repeat("a", 513)
		longRoles := []string{longRole}
		req := handler.Request{
			ApiId: apiID,
			Roles: &longRoles,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "validation")
	})
}
