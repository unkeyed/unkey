package principal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
)

func TestAuthorizeChecksPrincipalPermissions(t *testing.T) {
	t.Parallel()

	// Authorize must accept a principal whose permissions satisfy the query.
	p := &Principal{
		Version:               "",
		Subject:               Subject{ID: "", Name: "", Type: ""},
		Type:                  TypeAPIKey,
		Source:                KeySource{},
		AuthorizedWorkspaceID: "",
		Permissions:           []string{"api.*.create_api"},
	}

	err := p.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))

	require.NoError(t, err)
}

// TestAuthorizationErrorReturnsDenial guarantees request middleware can inspect
// the authorization error after a handler returns it.
func TestAuthorizationErrorReturnsDenial(t *testing.T) {
	t.Parallel()

	p := &Principal{
		Version:               "",
		Subject:               Subject{ID: "", Name: "", Type: ""},
		Type:                  TypeAPIKey,
		Source:                KeySource{},
		AuthorizedWorkspaceID: "",
		Permissions:           []string{},
	}

	err := p.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))

	require.Error(t, err)
	require.Equal(t, err, AuthorizationError(p))
}
