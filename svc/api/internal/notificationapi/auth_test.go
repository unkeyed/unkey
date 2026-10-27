package notificationapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
)

func TestPersonalPreferencesRequireAnExplicitUserSource(t *testing.T) {
	for _, tc := range []struct {
		name    string
		source  principal.Source
		subject principal.Subject
		allowed bool
	}{
		{name: "dashboard roles", source: principal.JWTSource{Roles: []string{"viewer"}}, subject: principal.Subject{Type: principal.SubjectTypeUser, ID: uid.New(uid.TestPrefix)}, allowed: true},
		{name: "cli user", source: principal.CLISessionSource{}, subject: principal.Subject{Type: principal.SubjectTypeUser, ID: uid.New(uid.TestPrefix)}, allowed: true},
		{name: "jwt without roles", source: principal.JWTSource{}, subject: principal.Subject{Type: principal.SubjectTypeUser, ID: uid.New(uid.TestPrefix)}},
		{name: "portal wildcard", source: principal.PortalSessionSource{}, subject: principal.Subject{Type: principal.SubjectTypeUser, ID: uid.New(uid.TestPrefix)}},
		{name: "root wildcard", source: principal.KeySource{}, subject: principal.Subject{Type: principal.SubjectTypeRootKey, ID: uid.New(uid.KeyPrefix)}},
		{name: "root source with user subject", source: principal.KeySource{}, subject: principal.Subject{Type: principal.SubjectTypeUser, ID: uid.New(uid.TestPrefix)}},
		{name: "empty user", source: principal.JWTSource{Roles: []string{"admin"}}, subject: principal.Subject{Type: principal.SubjectTypeUser}},
		{name: "whitespace user", source: principal.CLISessionSource{}, subject: principal.Subject{Type: principal.SubjectTypeUser, ID: " "}},
		{name: "no source", subject: principal.Subject{Type: principal.SubjectTypeUser, ID: uid.New(uid.TestPrefix)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspaceID := uid.New(uid.WorkspacePrefix)
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil), 0))
			session.SetPrincipal(&principal.Principal{
				Subject:               tc.subject,
				Source:                tc.source,
				AuthorizedWorkspaceID: workspaceID,
				Permissions:           []string{"unkey:v1:" + workspaceID + ":**#*"},
			})
			got, err := notificationapi.PersonalUser(session)
			if tc.allowed {
				require.NoError(t, err)
				require.Equal(t, tc.subject.ID, got.Subject.ID)
			} else {
				require.Error(t, err)
				require.Nil(t, got)
			}
		})
	}
}
