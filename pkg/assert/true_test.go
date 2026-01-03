package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestTrue(t *testing.T) {

	t.Run("value is true", func(t *testing.T) {
		err := assert.True(true)
		require.NoError(t, err)
	})

	t.Run("value is false", func(t *testing.T) {
		err := assert.True(false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected true but got false")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should be true"
		err := assert.True(false, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}
