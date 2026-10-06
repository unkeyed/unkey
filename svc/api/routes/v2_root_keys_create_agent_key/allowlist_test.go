package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/agentsignup"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_agent_key"
)

func sessionWith(t *testing.T, body string, p *principal.Principal) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v2/rootKeys.createAgentKey", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	sess.SetPrincipal(p)
	return sess
}

func TestCreateAgentKeyRejectsAdminRoleBeforeMinting(t *testing.T) {
	t.Parallel()

	p := &principal.Principal{
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: "user_admin", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: "ws_customer",
		Permissions:           []string{"unkey:v1:ws_customer:**#*"},
	}
	err := (&handler.Handler{}).Handle(context.Background(), sessionWith(t, `{}`, p))
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), code)
}

func TestCreateAgentKeyRejectsPermissionsOutsideAllowlistEvenForWildcardPrincipal(t *testing.T) {
	t.Parallel()

	const workspaceID = "ws_customer"
	p := &principal.Principal{
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: "user_admin", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{agentsignup.Role}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           []string{"unkey:v1:" + workspaceID + ":**#*"},
	}
	for _, permission := range []string{
		"unkey:v1:" + workspaceID + ":projects/*/keyspaces/*/keys/*#decrypt",
		"unkey:v1:" + workspaceID + ":**#*",
		"unkey:v1:" + workspaceID + ":rootKeys/*#write",
		"unkey:v1:ws_other:projects/*/keyspaces/*#read",
	} {
		t.Run(permission, func(t *testing.T) {
			t.Parallel()
			body := `{"permissions":["` + permission + `"]}`
			err := (&handler.Handler{}).Handle(context.Background(), sessionWith(t, body, p))
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
		})
	}
}
