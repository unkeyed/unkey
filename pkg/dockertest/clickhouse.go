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

var clickhouseCtr shared

// clickhouseContainerConfig returns the container configuration for ClickHouse.
func clickhouseContainerConfig() containerConfig {
	return containerConfig{
		Image:        clickhouseImage,
		ExposedPorts: []string{clickhousePort, clickhouseHTTPPort},
		WaitStrategy: NewTCPWait(clickhousePort),
		WaitTimeout:  60 * time.Second,
		Env: map[string]string{
			"CLICKHOUSE_USER":     clickhouseUser,
			"CLICKHOUSE_PASSWORD": clickhousePassword,
		},
		Cmd:         []string{},
		Tmpfs:       nil,
		SkipCleanup: false,
	}
}

// ClickHouse starts (or reuses) a shared ClickHouse container and returns a fresh
// ephemeral database for this test. The database is dropped when the test completes.
//
// The container starts on the first call in the process and is reused by all
// subsequent calls. Each call creates a unique database with loaded schema,
// ensuring complete isolation between tests.
func ClickHouse(t *testing.T) ClickHouseConfig {
	t.Helper()

	ctr := clickhouseCtr.get(t, clickhouseContainerConfig())

	port := ctr.Port(clickhousePort)
	baseDSN := fmt.Sprintf("clickhouse://%s:%s@%s:%s?secure=false&skip_verify=true&dial_timeout=10s",
		clickhouseUser, clickhousePassword, ctr.Host, port)

	// Connect to the default database first.
	opts, err := ch.ParseDSN(baseDSN)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()

	// Wait for ClickHouse to be ready.
	ctx := context.Background()
	require.Eventually(t, func() bool {
		return conn.Ping(ctx) == nil
	}, 60*time.Second, 500*time.Millisecond)

	// Create an ephemeral database for this test.
	dbName := fmt.Sprintf("test_%d", time.Now().UnixNano())
	err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName))
	require.NoError(t, err)

	// Connect to the ephemeral database and apply schema.
	dbDSN := fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s?secure=false&skip_verify=true&dial_timeout=10s",
		clickhouseUser, clickhousePassword, ctr.Host, port, dbName)

	dbOpts, err := ch.ParseDSN(dbDSN)
	require.NoError(t, err)

	dbConn, err := ch.Open(dbOpts)
	require.NoError(t, err)
	defer func() { require.NoError(t, dbConn.Close()) }()

	require.Eventually(t, func() bool {
		return dbConn.Ping(ctx) == nil
	}, 30*time.Second, 500*time.Millisecond)

	applyClickHouseSchema(t, ctx, dbConn)

	// Clean up: drop the database when the test finishes.
	t.Cleanup(func() {
		cleanupConn, cleanupErr := ch.Open(opts)
		if cleanupErr != nil {
			t.Logf("  ClickHouse cleanup: failed to connect: %v", cleanupErr)
			return
		}
		defer cleanupConn.Close() //nolint:errcheck
		cleanupErr = cleanupConn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
		if cleanupErr != nil {
			t.Logf("  ClickHouse cleanup: failed to drop database %s: %v", dbName, cleanupErr)
		}
	})

	return ClickHouseConfig{
		DSN: dbDSN,
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
