package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
)

// TestUnreadableIdentityIsNotFound protects internal and external identity IDs from enumeration.
func TestUnreadableIdentityIsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	// Create a root key with no permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	workspaceID := h.Resources().UserWorkspace.ID
	externalID := uid.New(uid.TestPrefix)

	identityID := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		Environment: "default",
		ExternalID:  externalID,
	}).ID
	otherIdentityID := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		Environment: "default",
		ExternalID:  uid.New(uid.TestPrefix),
	}).ID

	for _, credential := range []struct {
		name string
		key  string
	}{
		{name: "no read permission", key: rootKey},
		{name: "another identity", key: h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:projects/*/identities/%s#read", workspaceID, otherIdentityID))},
	} {
		t.Run(credential.name, func(t *testing.T) {
			headers.Set("Authorization", "Bearer "+credential.key)
			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Identity: uid.New(uid.IdentityPrefix)})
			require.Equal(t, http.StatusNotFound, missing.Status)
			for _, identity := range []string{externalID, identityID} {
				res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Identity: identity})
				require.Equal(t, missing.Status, res.Status)
				require.Equal(t, missing.Body.Error, res.Body.Error)
			}
		})
	}
}
