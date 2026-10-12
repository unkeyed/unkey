package principal

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
)

// TestAuthorizeChecksPrincipalPermissions guarantees that a matching grant allows
// the operation.
func TestAuthorizeChecksPrincipalPermissions(t *testing.T) {
	t.Parallel()

	p := &Principal{
		Permissions: []string{"unkey:v1:ws_123:projects/*#write"},
	}

	err := p.Authorize(rbac.U(urn.New().Workspace("ws_123").Project("*"), permissions.Write))

	require.NoError(t, err)
}

// TestAuthorizationErrorReturnsDenial guarantees request middleware can inspect
// the authorization error after a handler returns it. For example, denying API
// creation without a grant leaves that same error available to the middleware.
func TestAuthorizationErrorReturnsDenial(t *testing.T) {
	t.Parallel()

	p := &Principal{}
	err := p.Authorize(rbac.U(urn.New().Workspace("ws_123").Project("*"), permissions.Write))

	require.Error(t, err)
	require.Equal(t, err, AuthorizationError(p))
}

// TestAuthorizeUsesPrincipalGrants guarantees that only Principal.Permissions
// determine access, not raw key-source grants. For example, read without update
// denies And(read, update) but allows Or(read, update); a grant for another project
// cannot authorize a read. Every denial retains its insufficient-permissions code.
func TestAuthorizeUsesPrincipalGrants(t *testing.T) {
	workspace := urn.New().Workspace(uid.New(uid.WorkspacePrefix))
	resource := workspace.Project(uid.New(uid.ProjectPrefix))
	read := rbac.U(resource, permissions.Read)
	update := rbac.U(resource, permissions.Write)
	otherRead := rbac.U(workspace.Project(uid.New(uid.ProjectPrefix)), permissions.Read)
	wildcardRead := rbac.U(workspace.Project("*"), permissions.Read)
	otherWorkspaceRead := rbac.U(urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project("*"), permissions.Read)
	for _, tt := range []struct {
		name            string
		principalGrants []string
		keyGrants       []string
		query           rbac.PermissionQuery
		wantAllowed     bool
	}{
		{"nil grants deny", nil, nil, read, false},
		{"empty grants deny", []string{}, nil, read, false},
		{"source grant cannot authorize", nil, []string{read.Value}, read, false},
		{"principal grant authorizes without source grant", []string{read.Value}, []string{update.Value}, read, true},
		{"different resource denies", []string{otherRead.Value}, nil, read, false},
		{"wildcard grant authorizes concrete resource", []string{wildcardRead.Value}, nil, read, true},
		{"other workspace wildcard denies", []string{otherWorkspaceRead.Value}, nil, read, false},
		{"and denies missing second grant", []string{read.Value}, nil, rbac.And(read, update), false},
		{"and denies missing first grant", []string{update.Value}, nil, rbac.And(read, update), false},
		{"and allows both grants", []string{read.Value, update.Value}, nil, rbac.And(read, update), true},
		{"or allows first grant", []string{read.Value}, nil, rbac.Or(read, update), true},
		{"or allows second grant", []string{update.Value}, nil, rbac.Or(read, update), true},
		{"or denies when both grants are missing", nil, nil, rbac.Or(read, update), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &Principal{
				Source:      KeySource{Permissions: tt.keyGrants},
				Permissions: tt.principalGrants,
			}
			err := p.Authorize(tt.query)
			if tt.wantAllowed {
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

// TestAuthorizeEmptyAndAllowsWithoutPermissions guarantees the Boolean identity
// for AND: zero requirements are satisfied. A principal with no grants therefore
// passes And() and has no stored denial.
func TestAuthorizeEmptyAndAllowsWithoutPermissions(t *testing.T) {
	p := &Principal{}
	require.NoError(t, p.Authorize(rbac.And()))
	require.NoError(t, AuthorizationError(p))
}

// TestAuthorizeAllowsEveryAuthenticationMethod guarantees that
// authorization uses normalized grants regardless of the authentication method.
// For example, a JWT principal with a read grant can read without source claims.
func TestAuthorizeAllowsEveryAuthenticationMethod(t *testing.T) {
	read := rbac.U(urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix)), permissions.Read)
	for _, tt := range []struct {
		name     string
		authType Type
		source   Source
	}{
		{"api key", TypeAPIKey, KeySource{}},
		{"jwt", TypeJWT, JWTSource{}},
		{"portal", TypePortalSession, PortalSessionSource{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &Principal{
				Type:        tt.authType,
				Source:      tt.source,
				Permissions: []string{read.Value},
			}
			require.NoError(t, p.Authorize(read))
			require.NoError(t, AuthorizationError(p))
		})
	}
}

// TestAuthorizeRetainsMalformedQueryError guarantees that an invalid query remains
// an internal error, not a missing-grant denial. For example, PermissionQuery{}
// returns and stores UnexpectedError even when the principal has a read grant.
func TestAuthorizeRetainsMalformedQueryError(t *testing.T) {
	read := rbac.U(urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix)), permissions.Read)
	p := &Principal{Permissions: []string{read.Value}}
	err := p.Authorize(rbac.PermissionQuery{})
	require.Error(t, err)
	require.Same(t, err, AuthorizationError(p))
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Internal.UnexpectedError.URN(), code)
}

// TestAuthorizeLogsDenialContext guarantees that operators can identify the subject,
// customer workspace, required grant, actual grants, and error behind a denial.
// For example, a root key owned by another workspace logs the customer workspace,
// not the key owner. A later successful read must not emit another denial warning.
func TestAuthorizeLogsDenialContext(t *testing.T) {
	capture := loggertest.Install(t)
	workspaceID := uid.New(uid.WorkspacePrefix)
	resource := urn.New().Workspace(workspaceID).Project(uid.New(uid.ProjectPrefix))
	read := rbac.U(resource, permissions.Read)
	write := rbac.U(resource, permissions.Write)
	p := &Principal{
		Subject:               Subject{ID: "subject_a", Type: SubjectTypeRootKey},
		Type:                  TypeAPIKey,
		AuthorizedWorkspaceID: workspaceID,
		Source:                KeySource{WorkspaceID: "key_owner_workspace", Permissions: []string{write.Value}},
		Permissions:           []string{read.Value},
	}
	err := p.Authorize(write)
	require.Error(t, err)
	records := capture.Records()
	require.Len(t, records, 1)
	require.Equal(t, slog.LevelWarn, records[0].Level)
	require.Equal(t, "principal authorization denied", records[0].Message)
	attrs := loggertest.FlatAttrs(records[0])
	require.Equal(t, workspaceID, attrs["workspace_id"])
	require.Equal(t, "API_KEY", attrs["principal_type"])
	require.Equal(t, "rootkey", attrs["subject_type"])
	require.Equal(t, "subject_a", attrs["subject_id"])
	require.Equal(t, resource.String()+"#write", attrs["required_permissions"])
	require.Equal(t, []string{resource.String() + "#read"}, attrs["granted_permissions"])
	require.Same(t, err, attrs["error"])

	require.NoError(t, p.Authorize(read))
	require.Len(t, capture.Records(), 1)
}

// TestAuthorizationErrorRetainsLatestDenialAfterSuccess guarantees that a successful
// check does not erase the last denial needed by request middleware. For example,
// denied update, denied delete, then allowed read leaves the delete error stored.
func TestAuthorizationErrorRetainsLatestDenialAfterSuccess(t *testing.T) {
	resource := urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix))
	read := rbac.U(resource, permissions.Read)
	p := &Principal{Permissions: []string{read.Value}}
	require.NoError(t, AuthorizationError(p))

	updateErr := p.Authorize(rbac.U(resource, permissions.Write))
	require.Error(t, updateErr)
	require.Same(t, updateErr, AuthorizationError(p))

	deleteErr := p.Authorize(rbac.U(resource, permissions.Delete))
	require.Error(t, deleteErr)
	require.NotSame(t, updateErr, deleteErr)
	require.Same(t, deleteErr, AuthorizationError(p))

	require.NoError(t, p.Authorize(read))
	require.Same(t, deleteErr, AuthorizationError(p))
}

// TestAuthorizationErrorHidesGrantsAndCredentials guarantees that
// error text exposes the missing requirement, not the caller's grants or source
// details. For example, a denied read names the read permission but excludes JWT
// claims and signatures. This guarantee covers error messages, not warning logs.
func TestAuthorizationErrorHidesGrantsAndCredentials(t *testing.T) {
	resource := urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix))
	read := rbac.U(resource, permissions.Read)
	write := rbac.U(resource, permissions.Write)
	deletePermission := rbac.U(resource, permissions.Delete)
	for _, tt := range []struct {
		name   string
		source Source
	}{
		{"key", KeySource{KeyID: "private_key_id", Permissions: []string{"private_source_grant"}}},
		{"jwt", JWTSource{Header: map[string]any{"private_header": "secret"}, Payload: map[string]any{"private_claim": "secret"}, Signature: "private_signature"}},
		{"portal", PortalSessionSource{SessionID: "private_session_id"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &Principal{Source: tt.source, Permissions: []string{write.Value, deletePermission.Value}}
			err := p.Authorize(read)
			require.Error(t, err)
			denial := AuthorizationError(p)
			require.Same(t, err, denial)
			require.EqualError(t, denial, "Missing permission: '"+resource.String()+"#read': insufficient permissions")
			require.Equal(t, "Missing permission: '"+resource.String()+"#read'", fault.UserFacingMessage(denial))
		})
	}
}
