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
		Version:     "",
		Subject:     Subject{ID: "", Name: "", Type: ""},
		Type:        "",
		Source:      Source{Key: nil, JWT: nil, PortalSession: nil},
		WorkspaceID: "",
		Permissions: []string{"api.*.create_api"},
	}

	err := p.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))

	require.NoError(t, err)
}

func TestAuthorizeRejectsMissingPrincipal(t *testing.T) {
	t.Parallel()

	// Authorize must fail closed when called before authentication.
	var p *Principal

	err := p.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))

	require.Error(t, err)
}
