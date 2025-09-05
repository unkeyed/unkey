package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/assert"
)

func TestNotZero(t *testing.T) {
	t.Run("string: non-zero value passes", func(t *testing.T) {
		err := assert.NotZero("hello")
		require.NoError(t, err)
	})

	t.Run("string: zero value fails", func(t *testing.T) {
		err := assert.NotZero("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("string: zero value fails with custom message", func(t *testing.T) {
		err := assert.NotZero("", "Database connection required")
		require.Error(t, err)
		require.Contains(t, err.Error(), "Database connection required")
	})

	t.Run("int: non-zero value passes", func(t *testing.T) {
		err := assert.NotZero(42)
		require.NoError(t, err)
	})

	t.Run("int: zero value fails", func(t *testing.T) {
		err := assert.NotZero(0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("bool: true passes", func(t *testing.T) {
		err := assert.NotZero(true)
		require.NoError(t, err)
	})

	t.Run("bool: false (zero value) fails", func(t *testing.T) {
		err := assert.NotZero(false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})

	t.Run("pointer: non-nil passes", func(t *testing.T) {
		err := assert.NotZero(&struct{}{})
		require.NoError(t, err)
	})

	t.Run("pointer: nil (zero value) fails", func(t *testing.T) {
		err := assert.NotZero((*struct{})(nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})
}

// TestNotZero_Structs tests NotZero with struct types
func TestNotZero_Structs(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	t.Run("non-zero struct passes", func(t *testing.T) {
		config := Config{Host: "localhost", Port: 8080}
		err := assert.NotZero(config)
		require.NoError(t, err)
	})

	t.Run("partially zero struct passes", func(t *testing.T) {
		config := Config{Host: "localhost"}
		err := assert.NotZero(config)
		require.NoError(t, err)
	})

	t.Run("zero struct fails", func(t *testing.T) {
		config := Config{}
		err := assert.NotZero(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is zero/default")
	})
}
