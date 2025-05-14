package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/assert"
)

func TestInRange(t *testing.T) {

	t.Run("int value is in range", func(t *testing.T) {
		err := assert.InRange(5, 1, 10)
		require.NoError(t, err)
	})

	t.Run("int value is equal to min", func(t *testing.T) {
		err := assert.InRange(1, 1, 10)
		require.NoError(t, err)
	})

	t.Run("int value is equal to max", func(t *testing.T) {
		err := assert.InRange(10, 1, 10)
		require.NoError(t, err)
	})

	t.Run("int value is below min", func(t *testing.T) {
		err := assert.InRange(0, 1, 10)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is out of range")
	})

	t.Run("int value is above max", func(t *testing.T) {
		err := assert.InRange(11, 1, 10)
		require.Error(t, err)
	})

	t.Run("float value is in range", func(t *testing.T) {
		err := assert.InRange(5.5, 1.0, 10.0)
		require.NoError(t, err)
	})

	t.Run("float value is below min", func(t *testing.T) {
		err := assert.InRange(0.5, 1.0, 10.0)
		require.Error(t, err)
	})

	t.Run("float value is above max", func(t *testing.T) {
		err := assert.InRange(10.5, 1.0, 10.0)
		require.Error(t, err)
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should be in range"
		err := assert.InRange(11, 1, 10, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}

// FuzzInRange tests the InRange function with fuzzing
func FuzzInRange(f *testing.F) {
	// Seed with some examples
	f.Add(5, 1, 10)
	f.Add(0, 1, 10)
	f.Add(11, 1, 10)
	f.Add(1, 1, 10)
	f.Add(10, 1, 10)

	f.Fuzz(func(t *testing.T, val, min, max int) {
		err := assert.InRange(val, min, max)
		if val >= min && val <= max {
			if err != nil {
				t.Errorf("InRange(%d, %d, %d) should return nil error but got: %v", val, min, max, err)
			}
		} else {
			if err == nil {
				t.Errorf("InRange(%d, %d, %d) should return error but got nil", val, min, max)
			}
		}
	})
}
