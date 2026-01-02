package assert_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestSome(t *testing.T) {

	t.Run("returns nil when at least one assertion passes", func(t *testing.T) {
		// Create a mix of passing and failing assertions
		err1 := errors.New("error 1")
		var err2 error = nil // This assertion passes
		err3 := errors.New("error 3")

		// Some should return nil because err2 is nil
		err := assert.Some(err1, err2, err3)
		require.NoError(t, err)
	})

	t.Run("returns first error when all assertions fail", func(t *testing.T) {
		// Create multiple failing assertions
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		err3 := errors.New("error 3")

		// Some should return the first error (err1)
		err := assert.Some(err1, err2, err3)
		require.Error(t, err)
		require.Equal(t, err1, err)
	})

	t.Run("returns nil when the first assertion passes", func(t *testing.T) {
		// First assertion passes, others fail
		var err1 error = nil // This assertion passes
		err2 := errors.New("error 2")
		err3 := errors.New("error 3")

		// Some should return nil because err1 is nil
		err := assert.Some(err1, err2, err3)
		require.NoError(t, err)
	})

	t.Run("returns nil when the last assertion passes", func(t *testing.T) {
		// Last assertion passes, others fail
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		var err3 error = nil // This assertion passes

		// Some should return nil because err3 is nil
		err := assert.Some(err1, err2, err3)
		require.NoError(t, err)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		// No assertions provided
		err := assert.Some()
		require.NoError(t, err)
	})

	t.Run("works with actual assertions", func(t *testing.T) {
		// Test with real assertion functions
		result := assert.Some(
			assert.Equal(1, 2, "numbers should match"),
			assert.Equal("hello", "hello"), // This will pass
			assert.Equal(true, false, "boolean should match"),
		)
		require.NoError(t, result)

		// All assertions fail
		result = assert.Some(
			assert.Equal(1, 2, "numbers should match"),
			assert.Equal("hello", "world", "strings should match"),
			assert.Equal(true, false, "boolean should match"),
		)
		require.Error(t, result)
	})
}
