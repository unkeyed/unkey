package auditactor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestFromPrincipalRecordsCLISessionMeta(t *testing.T) {
	userID := uid.New(uid.TestPrefix)
	sessionID := uid.New(uid.CLISessionPrefix)
	actor := FromPrincipal(&principal.Principal{
		Subject: principal.Subject{ID: userID, Name: "Ada", Type: principal.SubjectTypeUser},
		Source:  principal.CLISessionSource{SessionID: sessionID, DeviceName: "laptop"},
	})
	require.Equal(t, "user", string(actor.Type))
	require.Equal(t, userID, actor.ID)
	require.Equal(t, "cli", actor.Meta["via"])
	require.Equal(t, sessionID, actor.Meta["session_id"])
	require.Equal(t, "laptop", actor.Meta["device"])
	require.NotContains(t, actor.Meta, "client_id")
}

// TestFromPrincipalRecordsTheOAuthClient guarantees that audit logs separate
// actions taken through an OAuth app from actions taken in the Unkey CLI, and
// name the app.
func TestFromPrincipalRecordsTheOAuthClient(t *testing.T) {
	sessionID := uid.New(uid.CLISessionPrefix)
	clientID := uid.New(uid.OAuthClientPrefix, 16)
	actor := FromPrincipal(&principal.Principal{
		Subject: principal.Subject{ID: uid.New(uid.TestPrefix), Name: "Ada", Type: principal.SubjectTypeUser},
		Source:  principal.CLISessionSource{SessionID: sessionID, DeviceName: "Unkey MCP", ClientID: clientID},
	})
	require.Equal(t, "oauth", actor.Meta["via"])
	require.Equal(t, clientID, actor.Meta["client_id"])
	require.Equal(t, sessionID, actor.Meta["session_id"])
	require.Equal(t, "Unkey MCP", actor.Meta["device"])
}
