package portalrbac_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auth/portalrbac"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

const (
	ws  = "ws_123"
	ks1 = "ks_111"
	ks2 = "ks_222"
)

// These queries mirror the URN legs the shared handlers authorize against, so a
// passing rbac.Check proves Expand's output actually satisfies them.
func listKeysQuery(ks string) rbac.PermissionQuery {
	return rbac.And(
		rbac.U(urn.New().Workspace(ws).Keyspace(ks).Key("*"), permissions.ReadKey{}),
		rbac.U(urn.New().Workspace(ws).Keyspace(ks), permissions.ReadKeyspace{}),
	)
}

func rerollQuery(ks string) rbac.PermissionQuery {
	return rbac.U(urn.New().Workspace(ws).Keyspace(ks), permissions.CreateKey{})
}

func analyticsQuery() rbac.PermissionQuery {
	return rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.ReadAnalytics})
}

func TestParseRejectsUnknownCapability(t *testing.T) {
	_, err := portalrbac.Parse("keys:destroy")
	require.Error(t, err)

	c, err := portalrbac.Parse("keys:reroll")
	require.NoError(t, err)
	require.Equal(t, portalrbac.CapKeysReroll, c)
}

func TestExpandSatisfiesHandlerQueries(t *testing.T) {
	granted := portalrbac.Grant{
		WorkspaceID:  ws,
		KeyspaceIDs:  []string{ks1},
		Capabilities: []portalrbac.Capability{portalrbac.CapKeysRead, portalrbac.CapKeysReroll, portalrbac.CapAnalyticsRead},
	}.Expand()

	require.NoError(t, rbac.Check(listKeysQuery(ks1), granted), "keys:read should satisfy listKeys")
	require.NoError(t, rbac.Check(rerollQuery(ks1), granted), "keys:reroll should satisfy reroll")
	require.NoError(t, rbac.Check(analyticsQuery(), granted), "analytics:read should satisfy getVerifications")
}

func TestExpandIsScopedToGrantedKeyspaces(t *testing.T) {
	granted := portalrbac.Grant{
		WorkspaceID:  ws,
		KeyspaceIDs:  []string{ks1},
		Capabilities: []portalrbac.Capability{portalrbac.CapKeysReroll},
	}.Expand()

	require.NoError(t, rbac.Check(rerollQuery(ks1), granted), "reroll allowed on granted keyspace")
	require.Error(t, rbac.Check(rerollQuery(ks2), granted), "reroll must be denied on a keyspace not in the grant")
}

func TestExpandDoesNotOvergrant(t *testing.T) {
	// A read-only grant must not satisfy a reroll (create_key) query.
	granted := portalrbac.Grant{
		WorkspaceID:  ws,
		KeyspaceIDs:  []string{ks1},
		Capabilities: []portalrbac.Capability{portalrbac.CapKeysRead},
	}.Expand()

	require.NoError(t, rbac.Check(listKeysQuery(ks1), granted))
	require.Error(t, rbac.Check(rerollQuery(ks1), granted), "read-only grant must not allow reroll")
	require.Error(t, rbac.Check(analyticsQuery(), granted), "read-only grant must not allow analytics")
}
