package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
)

func TestCanRead(t *testing.T) {
	t.Parallel()

	project := urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix))
	environmentID := uid.New(uid.EnvironmentPrefix)
	environment := project.App(uid.New(uid.AppPrefix)).Environment(environmentID)
	apiID := uid.New(uid.APIPrefix)
	keys := project.Keyspace(uid.New(uid.KeySpacePrefix)).Key("*")
	otherKeys := project.Keyspace(uid.New(uid.KeySpacePrefix)).Key("*")
	deployment := environment.Deployment(uid.New(uid.DeploymentPrefix))

	writeKeys := U(keys, permissions.Write)
	readKeys := keys.String() + "#read"
	writeOnly := keys.String() + "#write"

	for _, tt := range []struct {
		name    string
		query   PermissionQuery
		granted []string
		want    bool
	}{
		{
			name:    "urn read satisfies a write denial",
			query:   writeKeys,
			granted: []string{readKeys},
			want:    true,
		},
		{
			name:    "urn write does not count as read",
			query:   writeKeys,
			granted: []string{writeOnly},
			want:    false,
		},
		{
			name:    "no grants cannot read",
			query:   writeKeys,
			granted: nil,
			want:    false,
		},
		{
			name:    "read on a different keyspace does not reveal this one",
			query:   writeKeys,
			granted: []string{otherKeys.String() + "#read"},
			want:    false,
		},
		{
			name:    "wildcard read covers a concrete resource",
			query:   U(deployment, permissions.Write),
			granted: []string{environment.Deployment("*").String() + "#read"},
			want:    true,
		},
		{
			name: "legacy read api does not reveal a create key denial",
			query: Or(
				T(Tuple{ResourceType: Api, ResourceID: apiID, Action: CreateKey}),
				T(Tuple{ResourceType: Api, ResourceID: "*", Action: CreateKey}),
			),
			granted: []string{"api.*.read_api"},
			want:    false,
		},
		{
			name: "legacy create key does not reveal the api",
			query: Or(
				T(Tuple{ResourceType: Api, ResourceID: "*", Action: UpdateKey}),
			),
			granted: []string{"api.*.create_key"},
			want:    false,
		},
		{
			name: "role read does not reveal a permission",
			query: Or(
				T(Tuple{ResourceType: Rbac, ResourceID: "*", Action: DeletePermission}),
			),
			granted: []string{"rbac.*.read_role"},
			want:    false,
		},
		{
			name: "legacy permission read does not reveal a permission delete",
			query: Or(
				T(Tuple{ResourceType: Rbac, ResourceID: "*", Action: DeletePermission}),
			),
			granted: []string{"rbac.*.read_permission"},
			want:    false,
		},
		{
			name: "legacy deployment read does not reveal a rollback",
			query: Or(
				T(Tuple{ResourceType: Environment, ResourceID: "*", Action: RollbackDeployment}),
				T(Tuple{ResourceType: Environment, ResourceID: environmentID, Action: RollbackDeployment}),
				U(deployment, permissions.Write),
			),
			granted: []string{"environment.*.read_deployment"},
			want:    false,
		},
		{
			name: "environment read reveals a rollback on that environment",
			query: Or(
				T(Tuple{ResourceType: Environment, ResourceID: "*", Action: RollbackDeployment}),
				U(environment, permissions.Write),
			),
			granted: []string{environment.String() + "#read"},
			want:    true,
		},
		{
			name: "read analytics does not reveal an api",
			query: Or(
				T(Tuple{ResourceType: Api, ResourceID: apiID, Action: CreateKey}),
			),
			granted: []string{"api.*.read_analytics"},
			want:    false,
		},
		{
			name: "read portal does not reveal a session grant",
			query: Or(
				T(Tuple{ResourceType: Portal, ResourceID: uid.New(uid.PortalPrefix), Action: CreatePortalSession}),
			),
			granted: []string{"portal.*.read_portal"},
			want:    false,
		},
		{
			name: "unrelated deployment action does not reveal the deployment",
			query: Or(
				T(Tuple{ResourceType: Environment, ResourceID: "*", Action: RollbackDeployment}),
				U(deployment, permissions.Write),
			),
			granted: []string{"environment.*.promote_deployment"},
			want:    false,
		},
		{
			name:    "deployment read reveals build logs",
			query:   U(deployment.BuildLogs(), permissions.Read),
			granted: []string{deployment.String() + "#read"},
			want:    true,
		},
		{
			name:    "environment read does not reveal a deployment",
			query:   U(deployment, permissions.Write),
			granted: []string{environment.String() + "#read"},
			want:    false,
		},
		{
			name:    "a failed read query is not readable",
			query:   U(keys, permissions.Read),
			granted: []string{writeOnly},
			want:    false,
		},
		{
			name: "and is readable only when every branch is",
			query: And(
				U(keys, permissions.Write),
				U(deployment, permissions.Write),
			),
			granted: []string{readKeys},
			want:    false,
		},
		{
			name: "or is readable when any branch is",
			query: Or(
				T(Tuple{ResourceType: Api, ResourceID: apiID, Action: CreateKey}),
				U(keys, permissions.Write),
			),
			granted: []string{readKeys},
			want:    true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, CanRead(tt.query, tt.granted))
		})
	}
}

func TestPermissionDeniedDetailNamesTheMissingPermission(t *testing.T) {
	t.Parallel()

	keys := urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix)).Keyspace(uid.New(uid.KeySpacePrefix)).Key("*")
	query := U(keys, permissions.Write)
	detail := PermissionDeniedDetail(query, []string{query.Value[:len(query.Value)-len("#write")] + "#read"})

	require.Contains(t, detail, "Missing permission: '"+query.Value+"'")
	require.Contains(t, detail, "Log in again with that permission, or grant it.")
	require.NotContains(t, detail, "#read")
}

func TestPermissionDeniedDetailNamesEveryAlternative(t *testing.T) {
	t.Parallel()

	apiID := uid.New(uid.APIPrefix)
	query := Or(
		T(Tuple{ResourceType: Api, ResourceID: "*", Action: CreateKey}),
		T(Tuple{ResourceType: Api, ResourceID: apiID, Action: CreateKey}),
	)
	detail := PermissionDeniedDetail(query, nil)

	require.Contains(t, detail, "Missing one of these permissions:")
	require.Contains(t, detail, "api.*.create_key")
	require.Contains(t, detail, "api."+apiID+".create_key")
	require.Contains(t, detail, "Log in again with one of those permissions, or grant it.")
}
