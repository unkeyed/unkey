package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/assert"
)

func TestNotEqual(t *testing.T) {

	t.Run("strings are not equal", func(t *testing.T) {
		err := assert.NotEqual("test", "different")
		require.NoError(t, err)
	})

	t.Run("strings are equal", func(t *testing.T) {
		err := assert.NotEqual("test", "test")
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected not equal")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "custom error message"
		err := assert.NotEqual("test", "test", message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})

	t.Run("integers are not equal", func(t *testing.T) {
		err := assert.NotEqual(42, 43)
		require.NoError(t, err)
	})

	t.Run("integers are equal", func(t *testing.T) {
		err := assert.NotEqual(42, 42)
		require.Error(t, err)
	})

	t.Run("booleans are not equal", func(t *testing.T) {
		err := assert.NotEqual(true, false)
		require.NoError(t, err)
	})

	t.Run("booleans are equal", func(t *testing.T) {
		err := assert.NotEqual(true, true)
		require.Error(t, err)
	})
}

// FuzzNotEqual tests the NotEqual function with fuzzing
func FuzzNotEqual(f *testing.F) {
	// Seed with some examples
	f.Add("hello", "world")
	f.Add("hello", "hello")
	f.Add("", "nonempty")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, a, b string) {
		err := assert.NotEqual(a, b)
		if a == b {
			if err == nil {
				t.Errorf("NotEqual(%q, %q) should return error but got nil", a, b)
			}
		} else {
			if err != nil {
				t.Errorf("NotEqual(%q, %q) should return nil error but got: %v", a, b, err)
			}
		}
	})
}
