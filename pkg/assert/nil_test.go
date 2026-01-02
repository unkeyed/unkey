package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestNil(t *testing.T) {

	t.Run("value is nil", func(t *testing.T) {
		var nilValue interface{} = nil
		err := assert.Nil(nilValue)
		require.NoError(t, err)
	})

	t.Run("value is not nil", func(t *testing.T) {
		notNilValue := "not nil"
		err := assert.Nil(notNilValue)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected nil")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "value should be nil"
		notNilValue := 42
		err := assert.Nil(notNilValue, message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}
