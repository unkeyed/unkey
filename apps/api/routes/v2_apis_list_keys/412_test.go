package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestPreconditionError(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:   h.Logger,
		DB:       h.DB,
		Keys:     h.Keys,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create API manually
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
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
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key", "api.*.read_api", "api.*.decrypt_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for API ID with special characters
	t.Run("Try decrypting key without opt-in", func(t *testing.T) {
		req := handler.Request{
			ApiId:   apiID,
			Decrypt: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
