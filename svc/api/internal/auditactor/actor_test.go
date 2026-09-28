package auditactor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
)

func TestFromPrincipalOIDCVercel(t *testing.T) {
	t.Parallel()

	actor := FromPrincipal(&principal.Principal{
		Subject: principal.Subject{
			ID:   "vco_4kWyCQtR6cPP",
			Name: "Production API",
			Type: principal.SubjectTypeOIDCVercel,
		},
		Source: principal.OIDCVercelSource{IntegrationID: "vco_4kWyCQtR6cPP"},
	})

	require.Equal(t, auditlog.OIDCVercelActor, actor.Type)
	require.Equal(t, "vco_4kWyCQtR6cPP", actor.ID)
	require.Equal(t, "Production API", actor.Name)
}
