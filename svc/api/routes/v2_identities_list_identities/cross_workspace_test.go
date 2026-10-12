package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

func TestCrossWorkspaceForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	// Workspaces: we'll use the default user workspace and create a second one
	workspaceA := h.Resources().UserWorkspace.ID
	workspaceB := h.CreateWorkspace().ID

	// Create root key for workspace A with full permissions
	rootKeyA := h.CreateRootKey(workspaceA, "identity.*.read_identity")
	headersA := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyA)},
	}

	// Create an identity in workspace B
	externalID := uid.New(uid.TestPrefix)
	h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceB,
		Environment: "default",
		ExternalID:  externalID,
	})

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
