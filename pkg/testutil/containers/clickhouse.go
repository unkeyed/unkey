package containers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
)

const (
	clickhousePort     = 9000
	clickhouseUser     = "default"
	clickhousePassword = "password"
)

// ClickHouseConfig holds connection information for the ClickHouse test container.
type ClickHouseConfig struct {
	// DSN is the connection string for connecting from the test runner.
	DSN string
}

// ClickHouse starts the shared Docker Compose ClickHouse service and returns connection info.
//
// The container is reused through the worktree's Docker Compose project.
func ClickHouse(t testing.TB) ClickHouseConfig {
	t.Helper()

	c := startService(t, "clickhouse")
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d?secure=false&skip_verify=true&dial_timeout=10s",
		clickhouseUser, clickhousePassword, "localhost", c.Port(t, clickhousePort))

	// Connect and apply schema
	clickhouseOpts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)

	conn, err := ch.Open(clickhouseOpts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	// Wait for ClickHouse to be ready to accept queries
	ctx := context.Background()
	require.Eventually(t, func() bool {
		return conn.Ping(ctx) == nil
	}, 60*time.Second, 500*time.Millisecond)

	applyClickHouseSchema(t, ctx, conn)

	return ClickHouseConfig{
		DSN: dsn,
	}
}

// applyClickHouseSchema loads and executes schema files from pkg/clickhouse/schema/.
func applyClickHouseSchema(t testing.TB, ctx context.Context, conn ch.Conn) {
	t.Helper()

	// Enable experimental features needed by schema (e.g., JSON type)
	experimentalSettings := []string{
		"SET allow_experimental_json_type = 1",
		"SET allow_experimental_object_type = 1",
	}
	for _, setting := range experimentalSettings {
		err := conn.Exec(ctx, setting)
		require.NoError(t, err, "failed to enable experimental setting: %s", setting)
	}

	// Apply schema files
	schemaDir := clickhouseSchemaDir()
	applySchemaFiles(t, ctx, conn, schemaDir)
}

func applySchemaFiles(t testing.TB, ctx context.Context, conn ch.Conn, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read ClickHouse schema directory %q: %v", dir, err)
	}

	// Sort files by name to ensure correct order (000_, 001_, etc.)
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, file := range files {
		schemaPath := filepath.Join(dir, file)
		schemaBytes, readErr := os.ReadFile(schemaPath)
		require.NoError(t, readErr, "failed to read schema file: %s", schemaPath)

		// Split by semicolon and execute each statement
		statements := splitSQLStatements(string(schemaBytes))
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			err = conn.Exec(ctx, stmt)
			if err != nil {
				// Some statements may fail for expected reasons:
				// - UNKNOWN_TABLE: references tables that don't exist (v1 migration views)
				// - TABLE_ALREADY_EXISTS: table was already created
				errStr := err.Error()
				if !strings.Contains(errStr, "UNKNOWN_TABLE") &&
					!strings.Contains(errStr, "TABLE_ALREADY_EXISTS") &&
					!strings.Contains(errStr, "already exists") {
					require.NoError(t, err, "failed to execute schema statement from %s: %s", file, stmt)
				}
			}
		}
	}
}

// splitSQLStatements splits SQL content by semicolons, handling multi-line statements.
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		// Skip comment-only lines
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		current.WriteString(line)
		current.WriteString("\n")

		// Check if line ends with semicolon (end of statement)
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && stmt != ";" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// Handle any remaining content without trailing semicolon
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		statements = append(statements, remaining)
	}

	return statements
}

// clickhouseSchemaDir returns the path to the ClickHouse schema directory.
func clickhouseSchemaDir() string {
	return dataPath("pkg", "clickhouse", "schema")
}
