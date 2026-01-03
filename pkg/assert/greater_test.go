package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestGreater(t *testing.T) {

	t.Run("int a is greater than b", func(t *testing.T) {
		err := assert.Greater(10, 5)
		require.NoError(t, err)
	})

	t.Run("int a is equal to b", func(t *testing.T) {
		err := assert.Greater(5, 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "value is not greater")
	})

	t.Run("int a is less than b", func(t *testing.T) {
		err := assert.Greater(3, 5)
		require.Error(t, err)
	})

	t.Run("float a is greater than b", func(t *testing.T) {
		err := assert.Greater(10.5, 5.5)
		require.NoError(t, err)
	})

	t.Run("float a is equal to b", func(t *testing.T) {
		err := assert.Greater(5.5, 5.5)
		require.Error(t, err)
	})

	t.Run("float a is less than b", func(t *testing.T) {
		err := assert.Greater(3.5, 5.5)
		require.Error(t, err)
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "a should be greater than b"
		err := assert.Greater(5, 10, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}
