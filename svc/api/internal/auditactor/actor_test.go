package auditactor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
)

func TestFromPrincipalRecordsCLISessionMeta(t *testing.T) {
	actor := FromPrincipal(&principal.Principal{
		Subject: principal.Subject{ID: "user_1", Name: "Ada", Type: principal.SubjectTypeUser},
		Source:  principal.CLISessionSource{SessionID: "clis_1", DeviceName: "laptop"},
	})
	require.Equal(t, "user", string(actor.Type))
	require.Equal(t, "user_1", actor.ID)
	require.Equal(t, "cli", actor.Meta["via"])
	require.Equal(t, "clis_1", actor.Meta["session_id"])
	require.Equal(t, "laptop", actor.Meta["device"])
}
