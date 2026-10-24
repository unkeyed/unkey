package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
)

func TestDeleteProtection(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.delete_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for deleting an API with delete protection enabled
	t.Run("delete protected API", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		err := db.Query.UpdateApiDeleteProtection(ctx, h.DB.RW(), db.UpdateApiDeleteProtectionParams{
			ApiID:            api.ID,
			DeleteProtection: sql.NullBool{Valid: true, Bool: true},
		})
		require.NoError(t, err)

		// Ensure API exists and has delete protection
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.True(t, apiBeforeDelete.DeleteProtection.Valid)
		require.True(t, apiBeforeDelete.DeleteProtection.Bool)

		// Attempt to delete the API
		req := handler.Request{
			ApiId: api.ID,
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
		require.Equal(t, "This API has delete protection enabled. Disable it before attempting to delete.", res.Body.Error.Detail)

		// Verify API was NOT deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.False(t, apiAfterDelete.DeletedAtM.Valid, "API should not have been deleted")
	})
}

// TestDeleteProtectionWithReplicaLag keeps the API and its keys live when only
// the primary has observed delete protection being enabled.
func TestDeleteProtectionWithReplicaLag(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		KeySpaceID:  api.KeyAuthID.String,
		WorkspaceID: api.WorkspaceID,
	})

	replicaConfig := containers.MySQLIsolated(t)
	replica, err := db.New(db.Config{
		PrimaryDSN: replicaConfig.DSN,
		Tags:       sqlcomment.Disabled(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, replica.Close()) })
	err = db.Query.InsertApi(ctx, replica.RW(), db.InsertApiParams{
		ID:          api.ID,
		Name:        api.Name,
		WorkspaceID: api.WorkspaceID,
		ProjectID:   api.ProjectID,
		AuthType:    api.AuthType,
		IpWhitelist: api.IpWhitelist,
		KeyAuthID:   api.KeyAuthID,
		CreatedAtM:  api.CreatedAtM,
	})
	require.NoError(t, err)
	err = db.Query.UpdateApiDeleteProtection(ctx, h.DB.RW(), db.UpdateApiDeleteProtectionParams{
		ApiID:            api.ID,
		DeleteProtection: sql.NullBool{Valid: true, Bool: true},
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:        &laggingReplicaDatabase{Database: h.DB, replica: replica.RO()},
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)
	rootKey := h.CreateRootKey(api.WorkspaceID, "api.*.delete_api")
	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status)

	apiAfter, err := db.Query.FindApiByID(ctx, h.DB.RW(), api.ID)
	require.NoError(t, err)
	require.False(t, apiAfter.DeletedAtM.Valid)
	keyspaceAfter, err := db.Query.FindKeySpaceByID(ctx, h.DB.RW(), api.KeyAuthID.String)
	require.NoError(t, err)
	require.False(t, keyspaceAfter.DeletedAtM.Valid)
	keyAfter, err := db.Query.FindKeyByID(ctx, h.DB.RW(), key.KeyID)
	require.NoError(t, err)
	require.False(t, keyAfter.DeletedAtM.Valid)
}

type laggingReplicaDatabase struct {
	db.Database
	replica *db.Replica
}

func (d *laggingReplicaDatabase) RO() *db.Replica {
	return d.replica
}
