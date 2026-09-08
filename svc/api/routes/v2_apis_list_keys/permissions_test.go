package handler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

const permTestAPIID = "api_test"

// permAllows reports whether the query is satisfied by exactly the given grants.
func permAllows(t *testing.T, query rbac.PermissionQuery, grants ...string) bool {
	t.Helper()

	result, err := rbac.New().EvaluatePermissions(query, grants)
	require.NoError(t, err)

	return result.Valid
}

// TestReadKeysPermissionsShape pins the exported requirement to the query tree
// this route authorizes against. The literal below is the tree written out in
// full, so a change to the builder that alters evaluation or the string used in
// authorization error messages fails here.
func TestReadKeysPermissionsShape(t *testing.T) {
	inline := rbac.And(
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   permTestAPIID,
				Action:       rbac.ReadKey,
			}),
		),
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadAPI,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   permTestAPIID,
				Action:       rbac.ReadAPI,
			}),
		),
	)

	built := handler.ReadKeysPermissions(permTestAPIID)

	require.Equal(t,
		rbac.FormatPermissionQuery(inline),
		rbac.FormatPermissionQuery(built),
	)
	require.Equal(t, inline, built)
}

// TestReadKeysPermissionsRequiresBothActions pins the conjunction. The portal
// session route borrows this requirement as its ceiling, so weakening it to
// read_key alone would silently widen what a portal session can be granted.
func TestReadKeysPermissionsRequiresBothActions(t *testing.T) {
	query := handler.ReadKeysPermissions(permTestAPIID)

	require.False(t, permAllows(t, query))
	require.False(t, permAllows(t, query, "api."+permTestAPIID+".read_key"))
	require.False(t, permAllows(t, query, "api."+permTestAPIID+".read_api"))
	require.False(t, permAllows(t, query, "api.*.read_key"))
	require.False(t, permAllows(t, query, "api.*.read_api"))

	require.True(t, permAllows(t, query, "api."+permTestAPIID+".read_key", "api."+permTestAPIID+".read_api"))
	require.True(t, permAllows(t, query, "api.*.read_key", "api.*.read_api"))
	require.True(t, permAllows(t, query, "api.*.read_key", "api."+permTestAPIID+".read_api"))

	// A different api's grants must not satisfy this api.
	require.False(t, permAllows(t, query, "api.api_other.read_key", "api.api_other.read_api"))
}
