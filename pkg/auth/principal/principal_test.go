package principal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
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

func TestAuthorizeUsesPrincipalGrants(t *testing.T) {
	read := rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api_a", Action: rbac.ReadAPI})
	update := rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api_a", Action: rbac.UpdateAPI})
	for _, tt := range []struct {
		name        string
		permissions []string
		source      []string
		query       rbac.PermissionQuery
		allowed     bool
	}{
		{"nil grants deny", nil, nil, read, false},
		{"empty grants deny", []string{}, nil, read, false},
		{"source grant cannot authorize", nil, []string{"api.api_a.read_api"}, read, false},
		{"principal grant authorizes without source grant", []string{"api.api_a.read_api"}, []string{"api.api_a.update_api"}, read, true},
		{"different resource denies", []string{"api.api_b.read_api"}, nil, read, false},
		{"and denies missing second grant", []string{"api.api_a.read_api"}, nil, rbac.And(read, update), false},
		{"and denies missing first grant", []string{"api.api_a.update_api"}, nil, rbac.And(read, update), false},
		{"and allows both grants", []string{"api.api_a.read_api", "api.api_a.update_api"}, nil, rbac.And(read, update), true},
		{"or allows first grant", []string{"api.api_a.read_api"}, nil, rbac.Or(read, update), true},
		{"or allows second grant", []string{"api.api_a.update_api"}, nil, rbac.Or(read, update), true},
		{"or denies neither grant", nil, nil, rbac.Or(read, update), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &Principal{
				Subject:               Subject{ID: "subject_a", Type: SubjectTypeRootKey},
				Type:                  TypeAPIKey,
				AuthorizedWorkspaceID: "customer_workspace",
				Source:                KeySource{WorkspaceID: "key_owner_workspace", Permissions: tt.source},
				Permissions:           tt.permissions,
			}
			err := p.Authorize(tt.query)
			if tt.allowed {
				require.NoError(t, err)
				require.NoError(t, AuthorizationError(p))
				return
			}
			require.Error(t, err)
			require.Same(t, err, AuthorizationError(p))
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
		})
	}
}

func TestAuthorizeEmptyAndAllowsWithoutPermissions(t *testing.T) {
	p := &Principal{}
	require.NoError(t, p.Authorize(rbac.And()))
	require.NoError(t, AuthorizationError(p))
}

func TestAuthorizationErrorRetainsLatestDenialAfterSuccess(t *testing.T) {
	p := &Principal{Permissions: []string{"api.api_a.read_api"}}
	require.NoError(t, AuthorizationError(p))
	first := p.Authorize(rbac.S("api.api_a.update_api"))
	require.Error(t, first)
	require.Same(t, first, AuthorizationError(p))
	second := p.Authorize(rbac.S("api.api_a.delete_api"))
	require.Error(t, second)
	require.NotSame(t, first, second)
	require.Same(t, second, AuthorizationError(p))
	require.NoError(t, p.Authorize(rbac.S("api.api_a.read_api")))
	require.Same(t, second, AuthorizationError(p))
}

func TestAuthorizationErrorDoesNotExposeGrantedPermissionsOrSource(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source Source
	}{
		{"key", KeySource{KeyID: "private_key_id", Permissions: []string{"private_source_grant"}}},
		{"jwt", JWTSource{Header: map[string]any{"private_header": "secret"}, Payload: map[string]any{"private_claim": "secret"}, Signature: "private_signature"}},
		{"portal", PortalSessionSource{SessionID: "private_session_id"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &Principal{Source: tt.source, Permissions: []string{"private_grant_one", "private_grant_two"}}
			err := p.Authorize(rbac.S("api.api_a.read_api"))
			require.Error(t, err)
			stored := AuthorizationError(p)
			require.Same(t, err, stored)
			require.EqualError(t, stored, "Missing permission: 'api.api_a.read_api': insufficient permissions")
			require.Equal(t, "Missing permission: 'api.api_a.read_api'", fault.UserFacingMessage(stored))
		})
	}
}
