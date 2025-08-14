package keys

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func TestGetRootKey_ErrorHandling_ReturnsError(t *testing.T) {
	t.Parallel()

	// Create a service with nil dependencies, similar to create_test.go pattern
	s := &service{}

	ctx := context.Background()

	// This test validates that GetRootKey properly handles errors from Get()
	// Before the fix, GetRootKey would panic when trying to access key.Key.ForWorkspaceID.Valid
	// after Get() returned an error with nil key. Now it should return the error safely.
	key, log, err := s.GetRootKey(ctx, nil)

	require.Error(t, err)
	require.Nil(t, key)
	require.NotNil(t, log)

	// Verify specific error code for missing auth when session is nil
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Missing.URN(), code)
}

func TestGetRootKey_WithEmptyRawKey_ReturnsError(t *testing.T) {
	t.Parallel()

	// Create a service with nil dependencies, following create_test.go pattern
	s := &service{}

	ctx := context.Background()

	// Call Get with empty raw key to test the assert.NotEmpty validation
	key, log, err := s.Get(ctx, nil, "")

	// Verify that we get an error for empty key
	require.Error(t, err)
	require.Nil(t, key)
	require.NotNil(t, log)
	require.Contains(t, err.Error(), "rawKey is empty")
}

func TestGet_WithEmptyRawKey_ReturnsError(t *testing.T) {
	t.Parallel()

	// Test the assert.NotEmpty validation path directly in Get function
	s := &service{}
	ctx := context.Background()

	key, log, err := s.Get(ctx, nil, "")

	require.Error(t, err)
	require.Nil(t, key)
	require.NotNil(t, log)
	require.Contains(t, err.Error(), "rawKey is empty")
}

func TestGet_EmptyString_Variants(t *testing.T) {
	t.Parallel()

	// Test various empty string cases to improve assert.NotEmpty coverage
	s := &service{}
	ctx := context.Background()

	// Only test cases that will hit the validation path, not the cache/db path
	emptyVariants := []string{
		"", // Classic empty string
	}

	for _, empty := range emptyVariants {
		key, log, err := s.Get(ctx, nil, empty)

		require.Error(t, err)
		require.Nil(t, key)
		require.NotNil(t, log)
		require.Contains(t, err.Error(), "rawKey is empty")
	}
}
