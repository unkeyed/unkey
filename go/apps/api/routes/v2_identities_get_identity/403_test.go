package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
	}

	h.Register(route)

	// Create a root key with no permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create test identity
	ctx := context.Background()
	tx, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := h.Resources().UserWorkspace.ID
	identityID := uid.New(uid.IdentityPrefix)
	otherIdentityID := uid.New(uid.IdentityPrefix)
	externalID := "test_user_403"

	// Insert test identity
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Insert another test identity
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          otherIdentityID,
		ExternalID:  "other_user_403",
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	t.Run("no permission to read any identity", func(t *testing.T) {
		// The rootKey has no permissions, so it should fail
		req := handler.Request{
			IdentityId: &identityID,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Regexp(t, regexp.MustCompile(`^Missing one of these permissions: \[.*\], have: \[.*\]$`), res.Body.Error.Detail)
	})

	t.Run("permission for specific identity only", func(t *testing.T) {
		// Create a root key with permission for only the other identity
		specificPermKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity."+otherIdentityID+".read_identity")
		specificHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", specificPermKey)},
		}

		// Try to access the first identity, should fail
		req := handler.Request{
			IdentityId: &identityID,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, specificHeaders, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Regexp(t, regexp.MustCompile(`^Missing one of these permissions: \[.*\], have: \[.*\]$`), res.Body.Error.Detail)

		// Try to access the permitted identity, should succeed
		req = handler.Request{
			IdentityId: &otherIdentityID,
		}
		successRes := testutil.CallRoute[handler.Request, handler.Response](h, route, specificHeaders, req)
		require.Equal(t, http.StatusOK, successRes.Status)
		require.Equal(t, otherIdentityID, successRes.Body.Data.Id)
	})

	t.Run("permission by external ID but not by ID", func(t *testing.T) {
		// Create a key with specific identity permission
		specificPermKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity."+otherIdentityID+".read_identity")
		specificHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", specificPermKey)},
		}

		// Try to use externalId when only having permission for specific identity IDs
		req := handler.Request{
			ExternalId: &externalID,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, specificHeaders, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Regexp(t, regexp.MustCompile(`^Missing one of these permissions: \[.*\], have: \[.*\]$`), res.Body.Error.Detail)
	})
}
