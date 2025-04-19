package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

type Identity struct {
	ID           string
	ExternalID   string
	RatelimitIds []string
}

// Helper function that creates a new identity with rate-limits and returns it
func newIdentity(t *testing.T, h *testutil.Harness, numberOfRatelimits int) Identity {
	identityID := uid.New(uid.IdentityPrefix)
	externalID := uid.New("test_external_id")

	err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Meta:        nil,
		CreatedAt:   time.Now().UnixMilli(),
		Environment: "default",
	})
	require.NoError(t, err)

	ratelimitIds := make([]string, 0)
	for range numberOfRatelimits {
		rateLimitID := uid.New(uid.RatelimitPrefix)
		err = db.Query.InsertIdentityRatelimit(t.Context(), h.DB.RW(), db.InsertIdentityRatelimitParams{
			ID:          rateLimitID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			IdentityID:  sql.NullString{String: identityID, Valid: true},
			Name:        "Requests",
			Limit:       15,
			Duration:    (time.Minute * 15).Milliseconds(),
			CreatedAt:   time.Now().UnixMilli(),
		})

		require.NoError(t, err)
		ratelimitIds = append(ratelimitIds, rateLimitID)
	}

	return Identity{
		ID:           identityID,
		ExternalID:   externalID,
		RatelimitIds: ratelimitIds,
	}
}

func TestDeleteIdentitySuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a identity via DB and delete it again
	t.Run("delete identity via db and identity id", func(t *testing.T) {
		newIdentity := newIdentity(t, h, 0)

		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.NoError(t, err)

		err = db.Query.DeleteIdentity(ctx, h.DB.RW(), newIdentity.ID)
		require.NoError(t, err)

		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.Equal(t, sql.ErrNoRows, err)
	})

	// Create a identity via DB and delete it again
	t.Run("delete identity ratelimits via db", func(t *testing.T) {
		numberOfRatelimits := 2
		newIdentity := newIdentity(t, h, numberOfRatelimits)

		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.NoError(t, err)

		rateLimits, err := db.Query.FindRatelimitsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: newIdentity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, rateLimits, numberOfRatelimits)

		err = db.Query.DeleteIdentity(ctx, h.DB.RW(), newIdentity.ID)
		require.NoError(t, err)

		err = db.Query.DeleteManyRatelimitsByIDs(ctx, h.DB.RW(), newIdentity.RatelimitIds)
		require.NoError(t, err)

		_, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.Equal(t, sql.ErrNoRows, err)

		ratelimits, err := db.Query.FindRatelimitsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: newIdentity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, ratelimits, 0)
	})

	t.Run("delete identity via identityID", func(t *testing.T) {
		newIdentity := newIdentity(t, h, 0)

		req := handler.Request{IdentityId: ptr.P(newIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("delete identity via identityID", func(t *testing.T) {
		newIdentity := newIdentity(t, h, 0)

		req := handler.Request{ExternalId: ptr.P(newIdentity.ExternalID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("delete identity with ratelimits", func(t *testing.T) {
		newIdentity := newIdentity(t, h, 2)

		req := handler.Request{IdentityId: ptr.P(newIdentity.ID)}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)

		_, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), newIdentity.ID)
		require.Equal(t, sql.ErrNoRows, err)

		ratelimits, err := db.Query.FindRatelimitsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: newIdentity.ID, Valid: true})
		require.NoError(t, err)
		require.Len(t, ratelimits, 0)
	})
}
