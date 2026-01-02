package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestEmpty(t *testing.T) {

	t.Run("string is empty", func(t *testing.T) {
		err := assert.Empty("")
		require.NoError(t, err)
	})

	t.Run("string is not empty", func(t *testing.T) {
		err := assert.Empty("not empty")
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is not empty")
	})

	t.Run("slice is empty", func(t *testing.T) {
		var emptySlice []any
		err := assert.Empty(emptySlice)
		require.NoError(t, err)
	})

	t.Run("slice is not empty", func(t *testing.T) {
		nonEmptySlice := []any{1, 2, 3}
		err := assert.Empty(nonEmptySlice)
		require.Error(t, err)
	})

	t.Run("map is empty", func(t *testing.T) {
		var emptyMap map[any]any
		err := assert.Empty(emptyMap)
		require.NoError(t, err)
	})

	t.Run("map is not empty", func(t *testing.T) {
		nonEmptyMap := map[any]any{"key": "value"}
		err := assert.Empty(nonEmptyMap)
		require.Error(t, err)
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should be empty"
		err := assert.Empty("not empty", message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}

// FuzzEmpty tests the Empty function with fuzzing
func FuzzEmpty(f *testing.F) {
	// Seed with some examples
	f.Add("")
	f.Add("a")
	f.Add("hello world")
	f.Add(" ")  // Just a space
	f.Add("\t") // Tab character
	f.Add("\n") // Newline character

	f.Fuzz(func(t *testing.T, s string) {
		err := assert.Empty(s)
		if len(s) == 0 {
			// Empty string should pass
			if err != nil {
				t.Errorf("Empty(%q) should return nil for empty string but got: %v", s, err)
			}
		} else {
			// Non-empty string should fail
			if err == nil {
				t.Errorf("Empty(%q) should return error for non-empty string but got nil", s)
			}
		}
	})
}
