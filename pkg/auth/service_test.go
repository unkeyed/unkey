package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type stubResolver struct {
	principal *Principal
	err       error
}

func (r stubResolver) Resolve(_ context.Context, _ *zen.Session) (*Principal, error) {
	return r.principal, r.err
}

func TestServiceAuthenticate_UsesFirstResolvedPrincipal(t *testing.T) {
	t.Parallel()

	// The service must stop at the first resolver that authenticates the request.
	want := &Principal{
		Version: PrincipalVersion,
		Subject: Subject{
			ID:   "user_123",
			Name: "Dashboard User",
			Type: SubjectTypeUser,
		},
		Type:        PrincipalTypeJWT,
		Source:      Source{Key: nil, JWT: nil, PortalSession: nil},
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.create_api"},
	}
	service := New(stubResolver{}, stubResolver{principal: want})

	principal, err := service.Authenticate(context.Background(), &zen.Session{})

	require.NoError(t, err)
	require.Equal(t, want, principal)
}

func TestServiceAuthorize_ChecksPrincipalPermissions(t *testing.T) {
	t.Parallel()

	// Authorize must accept a principal whose permissions satisfy the query.
	service := New()
	principal := &Principal{
		Version:     "",
		Subject:     Subject{ID: "", Name: "", Type: ""},
		Type:        "",
		Source:      Source{Key: nil, JWT: nil, PortalSession: nil},
		WorkspaceID: "",
		Permissions: []string{"api.*.create_api"},
	}

	err := service.Authorize(context.Background(), principal, rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))

	require.NoError(t, err)
}

func TestServiceAuthorize_RejectsMissingPrincipal(t *testing.T) {
	t.Parallel()

	// Authorize must fail closed when called before authentication.
	service := New()

	err := service.Authorize(context.Background(), nil, rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))

	require.Error(t, err)
}
