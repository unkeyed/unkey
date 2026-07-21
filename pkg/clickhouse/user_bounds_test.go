package clickhouse

import (
	"context"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
)

type profileCaptureConn struct {
	driver.Conn
	queries []string
}

func (c *profileCaptureConn) Exec(_ context.Context, query string, _ ...any) error {
	c.queries = append(c.queries, query)
	return nil
}

func TestAnalyticsResultRowsMaxForWorkspace(t *testing.T) {
	// Security guarantee: the ClickHouse profile uses the lower workspace limit without permitting zero or oversized defaults.
	require.Equal(t, int32(37), AnalyticsResultRowsMaxForWorkspace(37))
	require.Equal(t, int32(AnalyticsResultRowsMax), AnalyticsResultRowsMaxForWorkspace(10_000_000))
	require.Equal(t, int32(AnalyticsResultRowsMax), AnalyticsResultRowsMaxForWorkspace(0))
}

func TestConfigureUserIncludesResultAndComplexityBounds(t *testing.T) {
	// Security guarantee: ClickHouse applies row, byte, and AST caps before producing analytics results.
	conn := &profileCaptureConn{}
	client := &Client{conn: conn}

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

	profileSQL := conn.queries[len(conn.queries)-1]
	for _, setting := range []string{
		"max_result_rows = 10000",
		"max_result_bytes = 4194304",
		"result_overflow_mode = 'throw'",
		"max_ast_depth = 100",
		"max_ast_elements = 2000",
	} {
		require.True(t, strings.Contains(profileSQL, setting), "missing profile setting %q", setting)
	}
}
