package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestFalse(t *testing.T) {

	t.Run("value is false", func(t *testing.T) {
		err := assert.False(false)
		require.NoError(t, err)
	})

	t.Run("value is true", func(t *testing.T) {
		err := assert.False(true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected false but got true")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should be false"
		err := assert.False(true, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}
