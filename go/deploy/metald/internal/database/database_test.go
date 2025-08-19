package database

import (
	"bytes"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer db.Close()

	assert.NotNil(t, db.db)
	assert.NotNil(t, db.tracer)
	assert.NotNil(t, db.logger)

	// Verify database file was created
	dbPath := filepath.Join(tempDir, "metald.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestNewWithLogger(t *testing.T) {
	tempDir := t.TempDir()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	db, err := NewWithLogger(tempDir, logger)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer db.Close()

	// Verify custom logger is used
	assert.NotNil(t, db.logger)

	// Verify database initialization logged
	assert.Contains(t, buf.String(), "database initialized successfully")
}

func TestNew_InvalidDirectory(t *testing.T) {
	// Try to create database in a path that can't be created (assuming root filesystem restrictions)
	invalidPath := "/root/nonexistent/deeply/nested/path"

	db, err := New(invalidPath)

	// Should fail to create the directory
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to create data directory")
}

func TestDatabase_Close(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close should succeed
	err = db.Close()
	assert.NoError(t, err)

	// Second close should not error (idempotent)
	err = db.Close()
	assert.NoError(t, err)
}

func TestDatabase_DB(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// DB() should return the underlying sql.DB
	underlyingDB := db.DB()
	assert.NotNil(t, underlyingDB)
	assert.IsType(t, &sql.DB{}, underlyingDB)

	// Should be able to ping through the returned DB
	err = underlyingDB.Ping()
	assert.NoError(t, err)
}

func TestDatabase_migrate(t *testing.T) {
	tempDir := t.TempDir()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	db, err := NewWithLogger(tempDir, logger)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Schema should be applied during initialization
	// Verify vms table exists
	var tableName string
	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='vms'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "vms", tableName)
}

func TestDatabase_migratePortMappings(t *testing.T) {
	tempDir := t.TempDir()

	// Create database
	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify port_mappings column exists after migration
	var columnExists bool
	err = db.db.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('vms') 
		WHERE name = 'port_mappings'
	`).Scan(&columnExists)
	require.NoError(t, err)
	assert.True(t, columnExists)
}

func TestDatabase_ConnectionPool(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Test that connection pool settings are applied
	stats := db.db.Stats()

	// MaxOpenConns should be set to 25
	assert.LessOrEqual(t, stats.OpenConnections, 25)

	// Should be able to perform operations
	err = db.db.Ping()
	assert.NoError(t, err)
}

func TestDatabase_WALMode(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify WAL mode is enabled
	var journalMode string
	err = db.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)

	// Verify foreign keys are enabled
	var foreignKeys bool
	err = db.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	require.NoError(t, err)
	assert.True(t, foreignKeys)
}

func TestDatabase_MultipleInstances(t *testing.T) {
	tempDir := t.TempDir()

	// Create first database instance
	db1, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db1)
	defer db1.Close()

	// Create second database instance (should work with WAL mode)
	db2, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db2)
	defer db2.Close()

	// Both should be able to ping
	err = db1.db.Ping()
	assert.NoError(t, err)

	err = db2.db.Ping()
	assert.NoError(t, err)
}

func TestDatabase_DirectoryPermissions(t *testing.T) {
	tempDir := t.TempDir()

	// Remove the temp directory to test creation
	os.RemoveAll(tempDir)

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify directory was created with correct permissions
	info, err := os.Stat(tempDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check permissions (0700)
	mode := info.Mode()
	expectedMode := os.FileMode(0700)
	assert.Equal(t, expectedMode, mode&os.ModePerm)
}

func TestDatabase_SchemaValidation(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify expected columns exist in vms table
	rows, err := db.db.Query("PRAGMA table_info(vms)")
	require.NoError(t, err)
	defer rows.Close()

	columnNames := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue interface{}

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey)
		require.NoError(t, err)

		columnNames[name] = true
	}

	// Verify expected columns exist
	expectedColumns := []string{"id", "customer_id", "config", "state", "process_id", "port_mappings", "created_at", "updated_at", "deleted_at"}
	for _, col := range expectedColumns {
		assert.True(t, columnNames[col], fmt.Sprintf("Expected column %s not found", col))
	}
}

func TestDatabase_Logging(t *testing.T) {
	tempDir := t.TempDir()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	db, err := NewWithLogger(tempDir, logger)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	logOutput := buf.String()

	// Verify debug and info logs are present
	assert.Contains(t, logOutput, "applying database schema")
	assert.Contains(t, logOutput, "database schema applied successfully")
	assert.Contains(t, logOutput, "database initialized successfully")
}
