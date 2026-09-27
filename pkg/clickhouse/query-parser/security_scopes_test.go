package queryparser_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	chquery "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
)

func TestParser_SecurityScopesPreserveTupleAncestry(t *testing.T) {
	config := chquery.Config{
		WorkspaceID:   "ws_safe",
		TableAliases:  map[string]string{"logs_v1": "default.logs"},
		AllowedTables: []string{"default.logs"},
		SecurityScopes: []chquery.SecurityScope{
			{Filters: []chquery.SecurityFilter{
				{Column: "project_id", AllowedValues: []string{"proj_a"}},
				{Column: "deployment_id", AllowedValues: []string{"dep_a"}},
			}},
			{Filters: []chquery.SecurityFilter{
				{Column: "project_id", AllowedValues: []string{"proj_b"}},
				{Column: "deployment_id", AllowedValues: []string{"dep_b"}},
			}},
		},
	}

	output, err := chquery.NewParser(config).Parse(context.Background(), "SELECT * FROM logs_v1")
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM default.logs WHERE logs.workspace_id = 'ws_safe' AND (((logs.project_id IN ('proj_a') AND logs.deployment_id IN ('dep_a')) OR (logs.project_id IN ('proj_b') AND logs.deployment_id IN ('dep_b'))))", output)
	require.NotContains(t, output, "project_id IN ('proj_a', 'proj_b')")
}

func TestParser_SecurityScopesNilAndEmpty(t *testing.T) {
	newConfig := func(scopes []chquery.SecurityScope) chquery.Config {
		return chquery.Config{
			WorkspaceID:    "ws_safe",
			TableAliases:   map[string]string{"logs_v1": "default.logs"},
			AllowedTables:  []string{"default.logs"},
			SecurityScopes: scopes,
		}
	}

	t.Run("nil adds no scope restriction", func(t *testing.T) {
		output, err := chquery.NewParser(newConfig(nil)).Parse(context.Background(), "SELECT * FROM logs_v1")
		require.NoError(t, err)
		require.Equal(t, "SELECT * FROM default.logs WHERE logs.workspace_id = 'ws_safe'", output)
	})

	for name, test := range map[string]struct {
		scopes []chquery.SecurityScope
		want   string
	}{
		"no resolved scopes": {
			scopes: []chquery.SecurityScope{},
			want:   "SELECT * FROM default.logs WHERE logs.workspace_id = 'ws_safe' AND ((0))",
		},
		"empty scope": {
			scopes: []chquery.SecurityScope{{Filters: []chquery.SecurityFilter{}}},
			want:   "SELECT * FROM default.logs WHERE logs.workspace_id = 'ws_safe' AND (((0)))",
		},
		"empty values": {
			scopes: []chquery.SecurityScope{{Filters: []chquery.SecurityFilter{
				{Column: "project_id", AllowedValues: []string{}},
			}}},
			want: "SELECT * FROM default.logs WHERE logs.workspace_id = 'ws_safe' AND (((0)))",
		},
	} {
		t.Run(name, func(t *testing.T) {
			output, err := chquery.NewParser(newConfig(test.scopes)).Parse(context.Background(), "SELECT * FROM logs_v1")
			require.NoError(t, err)
			require.Equal(t, test.want, output)
		})
	}
}

func TestParser_SecurityScopesCoexistWithFiltersAndCallerWhere(t *testing.T) {
	config := chquery.Config{
		WorkspaceID:     "ws_safe",
		TableAliases:    map[string]string{"logs_v1": "default.logs"},
		AllowedTables:   []string{"default.logs"},
		SecurityFilters: []chquery.SecurityFilter{{Column: "region", AllowedValues: []string{"eu"}}},
		SecurityScopes: []chquery.SecurityScope{{Filters: []chquery.SecurityFilter{
			{Column: "project_id", AllowedValues: []string{"proj_a"}},
		}}},
	}

	output, err := chquery.NewParser(config).Parse(context.Background(), "SELECT * FROM logs_v1 WHERE project_id = 'proj_forbidden' OR 1=1")
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM default.logs WHERE logs.workspace_id = 'ws_safe' AND (logs.region IN ('eu') AND (((logs.project_id IN ('proj_a'))) AND (project_id = 'proj_forbidden' OR 1 = 1)))", output)
}

func TestParser_SecurityScopesCoverEveryPhysicalSource(t *testing.T) {
	config := chquery.Config{
		WorkspaceID:   "ws_safe",
		TableAliases:  map[string]string{"logs_v1": "default.logs", "other_v1": "default.other"},
		AllowedTables: []string{"default.logs", "default.other"},
		SecurityScopes: []chquery.SecurityScope{{Filters: []chquery.SecurityFilter{
			{Column: "project_id", AllowedValues: []string{"proj_safe"}},
		}}},
	}
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{name: "alias", query: "SELECT * FROM logs_v1 l", want: []string{"l.project_id IN ('proj_safe')"}},
		{name: "join", query: "SELECT * FROM logs_v1 l CROSS JOIN other_v1 o", want: []string{"l.project_id IN ('proj_safe')", "o.project_id IN ('proj_safe')"}},
		{name: "CTE", query: "WITH scoped AS (SELECT * FROM logs_v1) SELECT * FROM scoped", want: []string{"logs.project_id IN ('proj_safe')"}},
		{name: "UNION", query: "SELECT * FROM logs_v1 l UNION ALL SELECT * FROM other_v1 o", want: []string{"l.project_id IN ('proj_safe')", "o.project_id IN ('proj_safe')"}},
		{name: "EXCEPT", query: "SELECT * FROM logs_v1 l EXCEPT SELECT * FROM other_v1 o", want: []string{"l.project_id IN ('proj_safe')", "o.project_id IN ('proj_safe')"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := chquery.NewParser(config).Parse(context.Background(), test.query)
			require.NoError(t, err)
			for _, expected := range test.want {
				require.Contains(t, output, expected)
			}
		})
	}
}
