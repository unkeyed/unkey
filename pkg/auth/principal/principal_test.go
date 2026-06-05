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
		Type:        TypeAPIKey,
		Source:      KeySource{},
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
