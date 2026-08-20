package keyperms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/keyperms"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

const (
	testWorkspaceID = "ws_test"
	testKeyspaceID  = "ks_test"
	testAPIID       = "api_test"
)

func testScope() keyperms.Scope {
	return keyperms.Scope{
		WorkspaceID: testWorkspaceID,
		KeyspaceID:  testKeyspaceID,
		APIID:       testAPIID,
	}
}

// allows reports whether the query is satisfied by exactly the given grants.
func allows(t *testing.T, query rbac.PermissionQuery, grants ...string) bool {
	t.Helper()

	result, err := rbac.New().EvaluatePermissions(query, grants)
	require.NoError(t, err)

	return result.Valid
}

// keyspaceURNGrant is a canonical URN grant on the test keyspace. No builder
// emits a URN leaf, so this must never satisfy any of them.
func keyspaceURNGrant(action string) string {
	return urn.New().Workspace(testWorkspaceID).Keyspace(testKeyspaceID).String() + "#" + action
}

// TestReadKeysMatchesListKeysLegacyBranch pins the builder to the legacy branch
// it replaced in svc/api/routes/v2_apis_list_keys. The literal below is a
// verbatim copy of that branch as it was written inline.
func TestReadKeysMatchesListKeysLegacyBranch(t *testing.T) {
	inline := rbac.And(
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   testAPIID,
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
				ResourceID:   testAPIID,
				Action:       rbac.ReadAPI,
			}),
		),
	)

	built := keyperms.ReadKeys(testScope())

	require.Equal(t,
		rbac.FormatPermissionQuery(inline),
		rbac.FormatPermissionQuery(built),
	)
	require.Equal(t, inline, built)
}

// TestCreateKeyMatchesRerollLegacyArm pins the builder to the legacy arm of
// rerollPermissionQuery in svc/api/routes/v2_keys_reroll_key. That query keeps
// its URN leaf at the call site, so parity is against the tuple arm only.
func TestCreateKeyMatchesRerollLegacyArm(t *testing.T) {
	inline := rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   testAPIID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
	)

	built := keyperms.CreateKey(testScope())

	require.Equal(t,
		rbac.FormatPermissionQuery(inline),
		rbac.FormatPermissionQuery(built),
	)
	require.Equal(t, inline, built)
}

// TestRerollQueryShapeParity checks that nesting the builders inside the route's
// existing Or/And keeps the full reroll requirement identical in evaluation and
// in the string used for authorization error messages.
func TestRerollQueryShapeParity(t *testing.T) {
	urnCreate := rbac.U(
		urn.New().Workspace(testWorkspaceID).Keyspace(testKeyspaceID),
		permissions.CreateKey{},
	)
	urnEncrypt := rbac.U(
		urn.New().Workspace(testWorkspaceID).Keyspace(testKeyspaceID).Key("*"),
		permissions.EncryptKey{},
	)

	inlineCreate := rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   testAPIID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
		urnCreate,
	)
	inlineEncrypted := rbac.And(
		inlineCreate,
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   testAPIID,
				Action:       rbac.EncryptKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.EncryptKey,
			}),
			urnEncrypt,
		),
	)

	builtCreate := rbac.Or(keyperms.CreateKey(testScope()), urnCreate)
	builtEncrypted := rbac.And(
		builtCreate,
		rbac.Or(keyperms.EncryptKey(testScope()), urnEncrypt),
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
		{"api." + testAPIID + ".create_key"},
		{"api.*.create_key"},
		{keyspaceURNGrant("create_key")},
		{"api." + testAPIID + ".create_key", "api." + testAPIID + ".encrypt_key"},
		{"api.*.create_key", "api.*.encrypt_key"},
		{"api." + testAPIID + ".encrypt_key"},
	}
	for _, grants := range grantSets {
		require.Equal(t,
			allows(t, inlineCreate, grants...),
			allows(t, builtCreate, grants...),
			"create arm diverged for grants %v", grants,
		)
		require.Equal(t,
			allows(t, inlineEncrypted, grants...),
			allows(t, builtEncrypted, grants...),
			"encrypted arm diverged for grants %v", grants,
		)
	}
}

func TestReadKeysRequiresBothActions(t *testing.T) {
	query := keyperms.ReadKeys(testScope())

	require.False(t, allows(t, query))
	require.False(t, allows(t, query, "api."+testAPIID+".read_key"))
	require.False(t, allows(t, query, "api."+testAPIID+".read_api"))
	require.False(t, allows(t, query, "api.*.read_key"))
	require.False(t, allows(t, query, "api.*.read_api"))

	require.True(t, allows(t, query, "api."+testAPIID+".read_key", "api."+testAPIID+".read_api"))
	require.True(t, allows(t, query, "api.*.read_key", "api.*.read_api"))
	require.True(t, allows(t, query, "api.*.read_key", "api."+testAPIID+".read_api"))

	// A different api's grants must not satisfy this scope.
	require.False(t, allows(t, query, "api.api_other.read_key", "api.api_other.read_api"))
}

func TestCreateKeyGrants(t *testing.T) {
	query := keyperms.CreateKey(testScope())

	require.True(t, allows(t, query, "api.*.create_key"))
	require.True(t, allows(t, query, "api."+testAPIID+".create_key"))
	require.False(t, allows(t, query, "api.api_other.create_key"))
	require.False(t, allows(t, query, "api.*.update_key"))

	// No builder emits a URN leaf, so a keyspace URN grant never satisfies one.
	require.False(t, allows(t, query, keyspaceURNGrant("create_key")))
}

func TestEncryptKeyIsSeparateFromCreateKey(t *testing.T) {
	create := keyperms.CreateKey(testScope())
	encrypt := keyperms.EncryptKey(testScope())

	require.NotEqual(t, create, encrypt)

	// Neither implies the other.
	require.False(t, allows(t, encrypt, "api.*.create_key"))
	require.False(t, allows(t, create, "api.*.encrypt_key"))

	require.True(t, allows(t, encrypt, "api.*.encrypt_key"))
	require.True(t, allows(t, encrypt, "api."+testAPIID+".encrypt_key"))
	require.False(t, allows(t, encrypt, keyspaceURNGrant("encrypt_key")))
}

func TestReadAnalyticsGrants(t *testing.T) {
	query := keyperms.ReadAnalytics(testScope())

	require.True(t, allows(t, query, "api.*.read_analytics"))
	require.True(t, allows(t, query, "api."+testAPIID+".read_analytics"))
	require.False(t, allows(t, query, "api.api_other.read_analytics"))
	require.False(t, allows(t, query))
	require.False(t, allows(t, query, keyspaceURNGrant("read_analytics")))
}

// TestFindApisByKeyAuthIds covers the keyspace-to-api reverse mapping the
// builders depend on for their Scope.APIID.
func TestFindApisByKeyAuthIds(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	workspace := h.Resources().UserWorkspace
	otherWorkspace := h.CreateWorkspace()

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	require.True(t, api.KeyAuthID.Valid)

	otherAPI := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   otherWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	require.True(t, otherAPI.KeyAuthID.Valid)

	rows, err := db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{api.KeyAuthID.String},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, api.KeyAuthID.String, rows[0].KeyAuthID)
	require.Equal(t, api.ID, rows[0].ApiID)

	// Workspace-scoped: another workspace's keyspace resolves to nothing.
	rows, err = db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{otherAPI.KeyAuthID.String},
	})
	require.NoError(t, err)
	require.Empty(t, rows)

	// Unknown keyspace ids are simply absent from the result.
	rows, err = db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{api.KeyAuthID.String, "ks_does_not_exist"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, api.ID, rows[0].ApiID)
}

// TestFindKeyAuthsByIdsAndWorkspaceReportsEncryption covers the column added to
// the keyspace batch query so a keyspace-wide caller can decide whether the
// encrypt-key arm applies.
func TestFindKeyAuthsByIdsAndWorkspaceReportsEncryption(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	workspace := h.Resources().UserWorkspace

	plain := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	encrypted := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: true,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	rows, err := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, h.DB.RO(), db.FindKeyAuthsByIdsAndWorkspaceParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{plain.KeyAuthID.String, encrypted.KeyAuthID.String},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byID := make(map[string]bool, len(rows))
	for _, row := range rows {
		byID[row.ID] = row.StoreEncryptedKeys
	}
	require.False(t, byID[plain.KeyAuthID.String])
	require.True(t, byID[encrypted.KeyAuthID.String])
}
