package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestGreaterOrEqual(t *testing.T) {

	t.Run("int a is greater than b", func(t *testing.T) {
		err := assert.GreaterOrEqual(10, 5)
		require.NoError(t, err)
	})

	t.Run("int a is equal to b", func(t *testing.T) {
		err := assert.GreaterOrEqual(5, 5)
		require.NoError(t, err)
	})

	t.Run("int a is less than b", func(t *testing.T) {
		err := assert.GreaterOrEqual(3, 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is not greater or equal")
	})

	t.Run("float a is greater than b", func(t *testing.T) {
		err := assert.GreaterOrEqual(10.5, 5.5)
		require.NoError(t, err)
	})

	t.Run("float a is equal to b", func(t *testing.T) {
		err := assert.GreaterOrEqual(5.5, 5.5)
		require.NoError(t, err)
	})

	t.Run("float a is less than b", func(t *testing.T) {
		err := assert.GreaterOrEqual(3.5, 5.5)
		require.Error(t, err)
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "a should be greater than or equal to b"
		err := assert.GreaterOrEqual(5, 10, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}

// FuzzGreaterOrEqual tests the GreaterOrEqual function with fuzzing
func FuzzGreaterOrEqual(f *testing.F) {
	// Seed with some examples
	f.Add(int32(10), int32(5))    // a > b
	f.Add(int32(5), int32(5))     // a = b
	f.Add(int32(3), int32(5))     // a < b
	f.Add(int32(-5), int32(-10))  // negative a > b
	f.Add(int32(-10), int32(-10)) // negative a = b
	f.Add(int32(-15), int32(-10)) // negative a < b

	f.Fuzz(func(t *testing.T, a, b int32) {
		err := assert.GreaterOrEqual(a, b)

		if a >= b {
			// Should pass
			if err != nil {
				t.Errorf("GreaterOrEqual(%d, %d) should return nil when a >= b, but got: %v", a, b, err)
			}
		} else {
			// Should fail
			if err == nil {
				t.Errorf("GreaterOrEqual(%d, %d) should return error when a < b, but got nil", a, b)
			}
		}
	})
}
