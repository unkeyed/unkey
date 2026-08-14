package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditPermission_Renders(t *testing.T) {
	tuple := Tuple{ResourceType: Audit, ResourceID: "*", Action: ReadAuditLog}
	require.Equal(t, "audit.*.read_audit_log", tuple.String())
}

func TestAuditPermission_Grantable(t *testing.T) {
	query := T(Tuple{ResourceType: Audit, ResourceID: "*", Action: ReadAuditLog})
	rbac := New()

	valid, err := rbac.EvaluatePermissions(query, []string{"audit.*.read_audit_log"})
	require.NoError(t, err)
	require.True(t, valid.Valid)

	denied, err := rbac.EvaluatePermissions(query, []string{"api.api1.read_api"})
	require.NoError(t, err)
	require.False(t, denied.Valid)
}
