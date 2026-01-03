package assert_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestAll(t *testing.T) {
	t.Run("returns nil when all assertions pass", func(t *testing.T) {
		// All assertions pass
		var err1 error = nil
		var err2 error = nil
		var err3 error = nil

		// All should return nil because all errors are nil
		err := assert.All(err1, err2, err3)
		require.NoError(t, err)
	})

	t.Run("returns first error when any assertion fails", func(t *testing.T) {
		// First assertion fails
		err1 := errors.New("error 1")
		var err2 error = nil
		var err3 error = nil

		// All should return the first error (err1)
		err := assert.All(err1, err2, err3)
		require.Error(t, err)
		require.Equal(t, err1, err)
	})

	t.Run("returns middle error when middle assertion fails", func(t *testing.T) {
		// Middle assertion fails
		var err1 error = nil
		err2 := errors.New("error 2")
		var err3 error = nil

		// All should return the first error encountered (err2)
		err := assert.All(err1, err2, err3)
		require.Error(t, err)
		require.Equal(t, err2, err)
	})

	t.Run("returns last error when last assertion fails", func(t *testing.T) {
		// Last assertion fails
		var err1 error = nil
		var err2 error = nil
		err3 := errors.New("error 3")

		// All should return the first error encountered (err3)
		err := assert.All(err1, err2, err3)
		require.Error(t, err)
		require.Equal(t, err3, err)
	})

	t.Run("returns first error when multiple assertions fail", func(t *testing.T) {
		// Multiple assertions fail
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		err3 := errors.New("error 3")

		// All should return the first error (err1)
		err := assert.All(err1, err2, err3)
		require.Error(t, err)
		require.Equal(t, err1, err)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		// No assertions provided
		err := assert.All()
		require.NoError(t, err)
	})

	t.Run("works with actual assertions", func(t *testing.T) {
		// Test with real assertion functions - all pass
		result := assert.All(
			assert.Equal(1, 1),
			assert.Equal("hello", "hello"),
			assert.Equal(true, true),
		)
		require.NoError(t, result)

		// One assertion fails
		result = assert.All(
			assert.Equal(1, 1),
			assert.Equal("hello", "world", "strings should match"),
			assert.Equal(true, true),
		)
		require.Error(t, result)
	})
}
