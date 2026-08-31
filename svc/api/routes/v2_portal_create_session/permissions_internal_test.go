package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/urn"
)

const (
	permTestWorkspaceID = "ws_test"
	permTestKeyspaceID  = "ks_test"
	permTestAPIID       = "api_test"
)

// permAllows reports whether the query is satisfied by exactly the given grants.
func permAllows(t *testing.T, query rbac.PermissionQuery, grants ...string) bool {
	t.Helper()

	result, err := rbac.New().EvaluatePermissions(query, grants)
	require.NoError(t, err)

	return result.Valid
}

// TestReadAnalyticsPermissionsGrants pins the analytics requirement, which is
// defined locally because v2_analytics_get_verifications has no query object to
// borrow. The URN case is the important one: that endpoint never evaluates URNs,
// so a keyspace-scoped URN grant must not let a caller mint a capability it could
// not exercise itself.
func TestReadAnalyticsPermissionsGrants(t *testing.T) {
	query := readAnalyticsPermissions(permTestAPIID)

	require.True(t, permAllows(t, query, "api.*.read_analytics"))
	require.True(t, permAllows(t, query, "api."+permTestAPIID+".read_analytics"))
	require.False(t, permAllows(t, query, "api.api_other.read_analytics"))
	require.False(t, permAllows(t, query))

	keyspaceURNGrant := urn.New().Workspace(permTestWorkspaceID).Keyspace(permTestKeyspaceID).String() + "#read_analytics"
	require.False(t, permAllows(t, query, keyspaceURNGrant))
}
