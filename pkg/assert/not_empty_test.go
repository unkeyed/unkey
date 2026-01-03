package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestNotEmpty(t *testing.T) {

	t.Run("string is not empty", func(t *testing.T) {
		err := assert.NotEmpty("not empty")
		require.NoError(t, err)
	})

	t.Run("string is empty", func(t *testing.T) {
		err := assert.NotEmpty("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is empty")
	})

	t.Run("slice is not empty", func(t *testing.T) {
		nonEmptySlice := []any{1, 2, 3}
		err := assert.NotEmpty(nonEmptySlice)
		require.NoError(t, err)
	})

	t.Run("slice is empty", func(t *testing.T) {
		var emptySlice []any
		err := assert.NotEmpty(emptySlice)
		require.Error(t, err)
	})

	t.Run("map is not empty", func(t *testing.T) {
		nonEmptyMap := map[any]any{"key": "value"}
		err := assert.NotEmpty(nonEmptyMap)
		require.NoError(t, err)
	})

	t.Run("map is empty", func(t *testing.T) {
		var emptyMap map[any]any
		err := assert.NotEmpty(emptyMap)
		require.Error(t, err)
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should not be empty"
		err := assert.NotEmpty("", message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}

// FuzzNotEmpty tests the NotEmpty function with fuzzing
func FuzzNotEmpty(f *testing.F) {
	// Seed with some examples
	f.Add("")
	f.Add("a")
	f.Add("hello world")
	f.Add(" ")  // Just a space
	f.Add("\t") // Tab character
	f.Add("\n") // Newline character

	f.Fuzz(func(t *testing.T, s string) {
		err := assert.NotEmpty(s)
		if len(s) == 0 {
			// Empty string should fail
			if err == nil {
				t.Errorf("NotEmpty(%q) should return error for empty string but got nil", s)
			}
		} else {
			// Non-empty string should pass
			if err != nil {
				t.Errorf("NotEmpty(%q) should return nil for non-empty string but got: %v", s, err)
			}
		}
	})
}
