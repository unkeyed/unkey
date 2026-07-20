package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestParserDisallowJoinsAllowsSingleSourceAndUnion(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID:   "ws_123",
		AllowedTables: []string{"default.events"},
		DisallowJoins: true,
	})

	for _, query := range []string{
		"SELECT * FROM default.events",
		"SELECT * FROM default.events UNION ALL SELECT * FROM default.events",
	} {
		_, err := parser.Parse(context.Background(), query)
		require.NoError(t, err, query)
	}
}

// TestParserDisallowJoinsRejectsEveryNestedMultiSourceForm protects the route
// policy from being bypassed by placing a join below the outer SELECT.
func TestParserDisallowJoinsRejectsEveryNestedMultiSourceForm(t *testing.T) {
	queries := map[string]string{
		"inner join":              "SELECT * FROM default.events a INNER JOIN default.events b ON a.id = b.id",
		"left join":               "SELECT * FROM default.events a LEFT JOIN default.events b ON a.id = b.id",
		"right join":              "SELECT * FROM default.events a RIGHT JOIN default.events b ON a.id = b.id",
		"full join":               "SELECT * FROM default.events a FULL JOIN default.events b ON a.id = b.id",
		"cross join":              "SELECT * FROM default.events CROSS JOIN default.events",
		"comma join":              "SELECT * FROM default.events a, default.events b",
		"join in CTE":             "WITH joined AS (SELECT * FROM default.events a JOIN default.events b ON a.id = b.id) SELECT * FROM joined",
		"join in nested subquery": "SELECT * FROM (SELECT * FROM (SELECT * FROM default.events a JOIN default.events b ON a.id = b.id))",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			parser := NewParser(Config{WorkspaceID: "ws_123", AllowedTables: []string{"default.events"}, DisallowJoins: true})
			_, err := parser.Parse(context.Background(), query)
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.User.BadRequest.InvalidAnalyticsQuery.URN(), code)
			require.Equal(t, "JOIN queries are not supported", fault.UserFacingMessage(err))
		})
	}
}

func TestParserAllowsAllowedTableJoinByDefault(t *testing.T) {
	parser := NewParser(Config{WorkspaceID: "ws_123", AllowedTables: []string{"default.events"}})
	_, err := parser.Parse(context.Background(), "SELECT * FROM default.events a JOIN default.events b ON a.id = b.id")
	require.NoError(t, err)
}
