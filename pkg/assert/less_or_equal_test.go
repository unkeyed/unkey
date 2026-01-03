package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestLessOrEqual(t *testing.T) {

	t.Run("int a is less than b", func(t *testing.T) {
		err := assert.LessOrEqual(5, 10)
		require.NoError(t, err)
	})

	t.Run("int a is equal to b", func(t *testing.T) {
		err := assert.LessOrEqual(5, 5)
		require.NoError(t, err)
	})

	t.Run("int a is greater than b", func(t *testing.T) {
		err := assert.LessOrEqual(10, 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is not less or equal")
	})

	t.Run("float a is less than b", func(t *testing.T) {
		err := assert.LessOrEqual(5.5, 10.5)
		require.NoError(t, err)
	})

	t.Run("float a is equal to b", func(t *testing.T) {
		err := assert.LessOrEqual(5.5, 5.5)
		require.NoError(t, err)
	})

	t.Run("float a is greater than b", func(t *testing.T) {
		err := assert.LessOrEqual(10.5, 5.5)
		require.Error(t, err)
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "a should be less than or equal to b"
		err := assert.LessOrEqual(10, 5, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}
