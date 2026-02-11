package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

func TestCrossWorkspaceForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
	}

	h.Register(route)

	ctx := context.Background()

	// Workspaces: we'll use the default user workspace and create a second one
	workspaceA := h.Resources().UserWorkspace.ID
	workspaceB := uid.New(uid.WorkspacePrefix)

	// Create root key for workspace A with full permissions
	rootKeyA := h.CreateRootKey(workspaceA, "identity.*.read_identity")
	headersA := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyA)},
	}

	// Create a test identity in workspace B
	tx, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	defer func() {
		err := tx.Rollback()
		require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
	}()

	// First, create the workspace B
	err = db.Query.InsertWorkspace(ctx, tx, db.InsertWorkspaceParams{
		ID:           workspaceB,
		Name:         "Test Workspace B",
		Slug:         uid.New("slug"),
		CreatedAt:    time.Now().UnixMilli(),
		OrgID:        uid.New("org"),
		K8sNamespace: uid.DNS1035(),
	})
	require.NoError(t, err)

	// Create an identity in workspace B
	identityB := uid.New(uid.IdentityPrefix)
	externalID := "user_in_workspace_b"
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          identityB,
		ExternalID:  externalID,
		WorkspaceID: workspaceB,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	t.Run("cannot access identities from another workspace", func(t *testing.T) {
		// Create a specific identity search query for workspaceB's identity
		req := handler.Request{}

		// Make the request using the key from workspace A
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersA, req)

		// The request should succeed because the key is valid
		require.Equal(t, http.StatusOK, res.Status)

		// But we should not see any identities from workspace B in the results
		for _, identity := range res.Body.Data {
			require.NotEqual(t, identity.ExternalId, externalID, "Identity from workspace B should not be accessible with key from workspace A")
		}
	})

}
