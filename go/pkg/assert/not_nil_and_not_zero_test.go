package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/assert"
)

func TestNotNilAndNotZero(t *testing.T) {
	t.Run("string: valid non-empty value passes", func(t *testing.T) {
		err := assert.NotNilAndNotZero("hello")
		require.NoError(t, err)
	})

	t.Run("string: empty string fails with zero error", func(t *testing.T) {
		err := assert.NotNilAndNotZero("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("int: non-zero value passes", func(t *testing.T) {
		err := assert.NotNilAndNotZero(42)
		require.NoError(t, err)
	})

	t.Run("int: zero value fails", func(t *testing.T) {
		err := assert.NotNilAndNotZero(0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("pointer: valid pointer passes", func(t *testing.T) {
		value := "test"
		err := assert.NotNilAndNotZero(&value)
		require.NoError(t, err)
	})

	t.Run("pointer: nil pointer fails with zero error", func(t *testing.T) {
		var ptr *string
		err := assert.NotNilAndNotZero(ptr)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("interface: non-nil interface with value passes", func(t *testing.T) {
		var iface interface{} = "test"
		err := assert.NotNilAndNotZero(iface)
		require.NoError(t, err)
	})

	t.Run("interface: nil interface fails with nil error", func(t *testing.T) {
		var iface interface{}
		err := assert.NotNilAndNotZero(iface)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected not nil")
	})

	t.Run("with custom message for nil pointer", func(t *testing.T) {
		var ptr *string
		err := assert.NotNilAndNotZero(ptr, "Database connection required")
		require.Error(t, err)
		require.Contains(t, err.Error(), "Database connection required")
	})

	t.Run("with custom message for zero", func(t *testing.T) {
		err := assert.NotNilAndNotZero("", "Configuration must be provided")
		require.Error(t, err)
		require.Contains(t, err.Error(), "Configuration must be provided")
	})
}

// TestNotNilAndNotZero_Structs tests NotNilAndNotZero with struct types
func TestNotNilAndNotZero_Structs(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	t.Run("struct pointer: valid initialized struct passes", func(t *testing.T) {
		config := &Config{Host: "localhost", Port: 8080}
		err := assert.NotNilAndNotZero(config)
		require.NoError(t, err)
	})

	t.Run("struct pointer: nil pointer fails", func(t *testing.T) {
		var config *Config
		err := assert.NotNilAndNotZero(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("struct pointer: pointer to zero struct fails", func(t *testing.T) {
		config := &Config{} // Points to zero value struct
		err := assert.NotNilAndNotZero(config)
		require.NoError(t, err) // Pointer itself is not nil or zero, even if it points to zero struct
	})

	t.Run("struct value: non-zero struct passes", func(t *testing.T) {
		config := Config{Host: "localhost", Port: 8080}
		err := assert.NotNilAndNotZero(config)
		require.NoError(t, err)
	})

	t.Run("struct value: zero struct fails", func(t *testing.T) {
		config := Config{}
		err := assert.NotNilAndNotZero(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})
}

// Database interface for testing
type Database interface {
	Query(string) error
}

// MockDB implements Database for testing
type MockDB struct {
	connected bool
}

func (m MockDB) Query(string) error { return nil }

// TestNotNilAndNotZero_DatabaseExample shows realistic usage with database-like interfaces
func TestNotNilAndNotZero_DatabaseExample(t *testing.T) {
	t.Run("database interface: valid implementation passes", func(t *testing.T) {
		var db Database = MockDB{connected: true}
		err := assert.NotNilAndNotZero(db)
		require.NoError(t, err)
	})

	t.Run("database interface: nil interface fails", func(t *testing.T) {
		var db Database
		err := assert.NotNilAndNotZero(db)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected not nil")
	})
}
