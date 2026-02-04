package dockertest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
)

const (
	// Use same image as docker-compose for compatibility
	clickhouseImage    = "bitnamilegacy/clickhouse:25.6.4"
	clickhousePort     = "9000/tcp"
	clickhouseHTTPPort = "8123/tcp"
	clickhouseUser     = "default"
	clickhousePassword = "password"
)

// ClickHouseConfig holds connection information for a ClickHouse test container.
type ClickHouseConfig struct {
	// DSN is the connection string for connecting from the test runner.
	DSN string
}

// ClickHouse starts a ClickHouse container and applies the schema.
//
// The container is started with a random available port and the schema
// is loaded from pkg/clickhouse/schema/*.sql files in order.
// This function blocks until ClickHouse is accepting connections and
// the schema has been applied. Fails the test if Docker is unavailable
// or the container fails to start.
func ClickHouse(t *testing.T) ClickHouseConfig {
	t.Helper()

	ctr := startContainer(t, containerConfig{
		Image:        clickhouseImage,
		ExposedPorts: []string{clickhousePort, clickhouseHTTPPort},
		WaitStrategy: NewTCPWait(clickhousePort),
		WaitTimeout:  60 * time.Second,
		Env: map[string]string{
			"CLICKHOUSE_USER":     clickhouseUser,
			"CLICKHOUSE_PASSWORD": clickhousePassword,
		},
		Cmd:   []string{},
		Tmpfs: nil,
	})

	port := ctr.Port(clickhousePort)
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%s?secure=false&skip_verify=true&dial_timeout=10s",
		clickhouseUser, clickhousePassword, ctr.Host, port)

	// Connect and apply schema
	opts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	// Wait for ClickHouse to be ready to accept queries
	ctx := context.Background()
	require.Eventually(t, func() bool {
		return conn.Ping(ctx) == nil
	}, 60*time.Second, 500*time.Millisecond)

	// Apply schema files
	applyClickHouseSchema(t, ctx, conn)

	return ClickHouseConfig{
		DSN: dsn,
	}
}

// applyClickHouseSchema loads and executes schema files from pkg/clickhouse/schema/.
func applyClickHouseSchema(t *testing.T, ctx context.Context, conn ch.Conn) {
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

func applySchemaFiles(t *testing.T, ctx context.Context, conn ch.Conn, dir string) {
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
	// Try Bazel runfiles first
	if runfiles := os.Getenv("TEST_SRCDIR"); runfiles != "" {
		workspace := os.Getenv("TEST_WORKSPACE")
		if workspace != "" {
			candidate := filepath.Join(runfiles, workspace, "pkg", "clickhouse", "schema")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
		candidate := filepath.Join(runfiles, "_main", "pkg", "clickhouse", "schema")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	// Fall back to relative path from this file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Dir(filepath.Dir(currentFile))
	return filepath.Join(root, "clickhouse", "schema")
}
