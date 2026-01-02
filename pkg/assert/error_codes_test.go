package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestErrorCodes(t *testing.T) {

	t.Run("assertion errors have correct error message", func(t *testing.T) {
		// Create an assertion error
		err := assert.Equal(1, 2)
		require.Error(t, err)

		// We can't directly check the error code from the string representation
		// Just verify that we get an error with the expected message format
		require.Contains(t, err.Error(), "expected equal")
	})
}
