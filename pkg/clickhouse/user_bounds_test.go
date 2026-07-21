package clickhouse

import (
	"context"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
)

type profileConnectionRecorder struct {
	driver.Conn
	queries []string
}

func (c *profileConnectionRecorder) Exec(_ context.Context, query string, _ ...any) error {
	c.queries = append(c.queries, query)
	return nil
}

// Security guarantee: ClickHouse applies the workspace row quota plus byte and AST caps before producing analytics results.
func TestConfigureUserIncludesResultAndComplexityBounds(t *testing.T) {
	connection := &profileConnectionRecorder{}
	client := &Client{conn: connection}

	err := client.ConfigureUser(context.Background(), UserConfig{
		WorkspaceID:               "ws_test",
		Username:                  "ws_test",
		Password:                  "password",
		AllowedTables:             []string{"default.key_verifications_raw_v2"},
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       1000,
		MaxExecutionTimePerWindow: 1800,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       1_000_000_000,
		MaxQueryResultRows:        10_000_000,
		RetentionDays:             30,
	})
	require.NoError(t, err)

	profileSQL := connection.queries[len(connection.queries)-1]
	for _, setting := range []string{
		"max_result_rows = 10000000",
		"max_result_bytes = 4194304",
		"result_overflow_mode = 'throw'",
		"max_ast_depth = 100",
		"max_ast_elements = 2000",
	} {
		require.True(t, strings.Contains(profileSQL, setting), "missing profile setting %q", setting)
	}
}

// Security guarantee: an invalid workspace quota cannot disable the ClickHouse result row bound.
func TestConfigureUserRejectsDisabledResultRowBound(t *testing.T) {
	connection := &profileConnectionRecorder{}
	client := &Client{conn: connection}

	err := client.ConfigureUser(context.Background(), UserConfig{
		WorkspaceID:        "ws_test",
		Username:           "ws_test",
		MaxQueryResultRows: 0,
	})
	require.ErrorContains(t, err, "query result row limit must be positive")
	require.Empty(t, connection.queries)
}
