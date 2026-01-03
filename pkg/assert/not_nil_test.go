package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestNotNil(t *testing.T) {

	t.Run("value is not nil", func(t *testing.T) {
		notNilValue := "not nil"
		err := assert.NotNil(notNilValue)
		require.NoError(t, err)
	})

	t.Run("value is nil", func(t *testing.T) {
		var nilValue interface{} = nil
		err := assert.NotNil(nilValue)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected not nil")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should not be nil"
		var nilValue interface{} = nil
		err := assert.NotNil(nilValue, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}
