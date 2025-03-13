package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateIdentitySuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, "identity.*.create_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("insert identity", func(t *testing.T) {
		// Create a identity
		identityID := uid.New(uid.TestPrefix + "_" + uid.IdentityPrefix)
		externalTestID := uid.New("test_external_id")
		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalTestID,
			WorkspaceID: h.Resources.UserWorkspace.ID,
			Meta:        nil,
			CreatedAt:   time.Now().UnixMilli(),
			Environment: "default",
		})
		require.NoError(t, err)

		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), identityID)
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, externalTestID)
	})

	// Test creating a identity with no other information
	t.Run("create identity", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")
		req := handler.Request{ExternalId: externalTestID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.IdentityId)

		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), res.Body.IdentityId)
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)
	})

	// Test creating a identity with metadata
	t.Run("create identity with metadata", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")

		var someMetaValue interface{} = "example"
		meta := &map[string]*interface{}{"key": ptr.P(someMetaValue)}
		req := handler.Request{
			ExternalId: externalTestID + "_2",
			Meta:       meta,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.IdentityId)

		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), res.Body.IdentityId)
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)

		var dbMeta *map[string]*interface{}
		json.Unmarshal(identity.Meta, &dbMeta)
		require.Equal(t, dbMeta, meta)
	})

	// Test creating a identity with ratelimits
	t.Run("create identity with ratelimits", func(t *testing.T) {
		externalTestID := uid.New("test_external_id")
		req := handler.Request{
			ExternalId: externalTestID,
			Ratelimits: &[]struct {
				Duration int    "json:\"duration\""
				Limit    int    "json:\"limit\""
				Name     string "json:\"name\""
			}{
				{
					Duration: int(time.Minute.Milliseconds()),
					Limit:    100,
					Name:     "test",
				},
				{
					Duration: int(time.Minute.Milliseconds()),
					Limit:    200,
					Name:     "test2",
				},
			},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.IdentityId)

		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), res.Body.IdentityId)
		require.NoError(t, err)
		require.Equal(t, identity.ExternalID, req.ExternalId)

		rateLimits, err := db.Query.FindRatelimitsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: res.Body.IdentityId, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, 2)

		for _, ratelimit := range *req.Ratelimits {
			idx := slices.IndexFunc(rateLimits, func(c db.FindRatelimitsByIdentityIDRow) bool { return c.Name == ratelimit.Name })

			require.True(t, idx <= len(rateLimits)-1)
			require.Equal(t, int(rateLimits[idx].Duration), ratelimit.Duration)
			require.Equal(t, int(rateLimits[idx].Limit), ratelimit.Limit)
			require.Equal(t, rateLimits[idx].Name, ratelimit.Name)
		}
	})
}
