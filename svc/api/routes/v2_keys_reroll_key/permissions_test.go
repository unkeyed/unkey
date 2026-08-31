package handler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

const (
	permTestWorkspaceID = "ws_test"
	permTestProjectID   = "proj_test"
	permTestKeyspaceID  = "ks_test"
	permTestKeyID       = "key_test"
	permTestAPIID       = "api_test"
)

// permAllows reports whether the query is satisfied by exactly the given grants.
func permAllows(t *testing.T, query rbac.PermissionQuery, grants ...string) bool {
	t.Helper()

	result, err := rbac.New().EvaluatePermissions(query, grants)
	require.NoError(t, err)

	return result.Valid
}

// permKeyURNGrant is a canonical URN grant on the test key. Neither legacy
// requirement below emits a URN leaf, so this must never satisfy one on its own.
func permKeyURNGrant(action string) string {
	return urn.New().
		Workspace(permTestWorkspaceID).
		Project(permTestProjectID).
		Keyspace(permTestKeyspaceID).
		Key(permTestKeyID).
		String() + "#" + action
}

// TestCreateKeyPermissionsShape pins the exported tuple arm. This route Ors its
// own URN leaf on top, so parity here is against the tuple arm only.
func TestCreateKeyPermissionsShape(t *testing.T) {
	inline := rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   permTestAPIID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
	)

	built := handler.CreateKeyPermissions(permTestAPIID)

	require.Equal(t,
		rbac.FormatPermissionQuery(inline),
		rbac.FormatPermissionQuery(built),
	)
	require.Equal(t, inline, built)
}

// TestRerollQueryShapeParity checks that nesting the exported requirements inside
// this route's Or/And keeps the full reroll requirement identical in evaluation
// and in the string used for authorization error messages.
func TestRerollQueryShapeParity(t *testing.T) {
	urnWrite := rbac.U(
		urn.New().
			Workspace(permTestWorkspaceID).
			Project(permTestProjectID).
			Keyspace(permTestKeyspaceID).
			Key(permTestKeyID),
		permissions.Write{},
	)

	inlineCreate := rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   permTestAPIID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
		urnWrite,
	)
	inlineEncrypted := rbac.And(
		inlineCreate,
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   permTestAPIID,
				Action:       rbac.EncryptKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.EncryptKey,
			}),
			urnWrite,
		),
	)

	builtCreate := rbac.Or(handler.CreateKeyPermissions(permTestAPIID), urnWrite)
	builtEncrypted := rbac.And(
		builtCreate,
		rbac.Or(handler.EncryptKeyPermissions(permTestAPIID), urnWrite),
	)

	require.Equal(t,
		rbac.FormatPermissionQuery(inlineCreate),
		rbac.FormatPermissionQuery(builtCreate),
	)
	require.Equal(t,
		rbac.FormatPermissionQuery(inlineEncrypted),
		rbac.FormatPermissionQuery(builtEncrypted),
	)

	grantSets := [][]string{
		{},
		{"api." + permTestAPIID + ".create_key"},
		{"api.*.create_key"},
		{permKeyURNGrant("write_key")},
		{"api." + permTestAPIID + ".create_key", "api." + permTestAPIID + ".encrypt_key"},
		{"api.*.create_key", "api.*.encrypt_key"},
		{"api." + permTestAPIID + ".encrypt_key"},
	}
	for _, grants := range grantSets {
		require.Equal(t,
			permAllows(t, inlineCreate, grants...),
			permAllows(t, builtCreate, grants...),
			"create arm diverged for grants %v", grants,
		)
		require.Equal(t,
			permAllows(t, inlineEncrypted, grants...),
			permAllows(t, builtEncrypted, grants...),
			"encrypted arm diverged for grants %v", grants,
		)
	}
}

func TestCreateKeyPermissionsGrants(t *testing.T) {
	query := handler.CreateKeyPermissions(permTestAPIID)

	require.True(t, permAllows(t, query, "api.*.create_key"))
	require.True(t, permAllows(t, query, "api."+permTestAPIID+".create_key"))
	require.False(t, permAllows(t, query, "api.api_other.create_key"))
	require.False(t, permAllows(t, query, "api.*.update_key"))

	// The tuple arm emits no URN leaf, so a key URN grant never satisfies it.
	require.False(t, permAllows(t, query, permKeyURNGrant("write_key")))
}

// TestEncryptKeyPermissionsIsSeparateFromCreateKey pins the two as independent
// requirements. Portal session minting relies on that separation: it adds the
// encrypt arm only when the keyspace stores recoverable key material.
func TestEncryptKeyPermissionsIsSeparateFromCreateKey(t *testing.T) {
	create := handler.CreateKeyPermissions(permTestAPIID)
	encrypt := handler.EncryptKeyPermissions(permTestAPIID)

	require.NotEqual(t, create, encrypt)

	// Neither implies the other.
	require.False(t, permAllows(t, encrypt, "api.*.create_key"))
	require.False(t, permAllows(t, create, "api.*.encrypt_key"))

	require.True(t, permAllows(t, encrypt, "api.*.encrypt_key"))
	require.True(t, permAllows(t, encrypt, "api."+permTestAPIID+".encrypt_key"))
	require.False(t, permAllows(t, encrypt, permKeyURNGrant("write_key")))
}
