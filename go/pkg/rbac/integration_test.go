package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuery_Integration(t *testing.T) {
	rbac := New()

	// Test parsing and evaluation together
	t.Run("Parse and evaluate simple query", func(t *testing.T) {
		query, err := ParseQuery("api.key1.read_key")
		require.NoError(t, err)

		userPermissions := []string{"api.key1.read_key", "api.key1.update_key"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate complex query", func(t *testing.T) {
		query, err := ParseQuery("api.key1.read_key AND (ratelimit.ns1.limit OR ratelimit.ns2.limit)")
		require.NoError(t, err)

		userPermissions := []string{
			"api.key1.read_key",
			"ratelimit.ns1.limit",
		}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate failing query", func(t *testing.T) {
		query, err := ParseQuery("api.key1.read_key AND api.key1.delete_key")
		require.NoError(t, err)

		userPermissions := []string{"api.key1.read_key"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.False(t, result.Valid)
		require.Contains(t, result.Message, "Missing permission: 'api.key1.delete_key'")
	})

	t.Run("Parse and evaluate OR query", func(t *testing.T) {
		query, err := ParseQuery("api.key1.read_key OR api.key1.update_key")
		require.NoError(t, err)

		userPermissions := []string{"api.key1.update_key"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate precedence", func(t *testing.T) {
		query, err := ParseQuery("perm1 OR perm2 AND perm3")
		require.NoError(t, err)

		// This should be parsed as "perm1 OR (perm2 AND perm3)"
		// So having just perm1 should be enough
		userPermissions := []string{"perm1"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})
}
