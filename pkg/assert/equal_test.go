package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestEqual(t *testing.T) {

	t.Run("strings are equal", func(t *testing.T) {
		err := assert.Equal("test", "test")
		require.NoError(t, err)
	})

	t.Run("strings are not equal", func(t *testing.T) {
		err := assert.Equal("test", "different")
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected equal")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "custom error message"
		err := assert.Equal("test", "different", message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})

	t.Run("integers are equal", func(t *testing.T) {
		err := assert.Equal(42, 42)
		require.NoError(t, err)
	})

	t.Run("integers are not equal", func(t *testing.T) {
		err := assert.Equal(42, 43)
		require.Error(t, err)
	})

	t.Run("booleans are equal", func(t *testing.T) {
		err := assert.Equal(true, true)
		require.NoError(t, err)
	})

	t.Run("booleans are not equal", func(t *testing.T) {
		err := assert.Equal(true, false)
		require.Error(t, err)
	})
}

// FuzzEqual tests the Equal function with fuzzing
func FuzzEqual(f *testing.F) {
	// Seed with some examples
	f.Add("hello", "hello")
	f.Add("hello", "world")
	f.Add("", "")
	f.Add("", "nonempty")

	f.Fuzz(func(t *testing.T, a, b string) {
		err := assert.Equal(a, b)
		if a == b {
			if err != nil {
				t.Errorf("Equal(%q, %q) should return nil error but got: %v", a, b, err)
			}
		} else {
			if err == nil {
				t.Errorf("Equal(%q, %q) should return error but got nil", a, b)
			}
		}
	})
}
