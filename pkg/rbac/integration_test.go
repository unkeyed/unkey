package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuery_Integration(t *testing.T) {
	rbac := New()

	// Test parsing and evaluation together
	t.Run("Parse and evaluate simple query", func(t *testing.T) {
		query, err := ParseQuery("documents.read")
		require.NoError(t, err)

		userPermissions := []string{"documents.read", "documents.write"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate complex query", func(t *testing.T) {
		query, err := ParseQuery("documents.read AND (billing.admin OR reports.export)")
		require.NoError(t, err)

		userPermissions := []string{
			"documents.read",
			"billing.admin",
		}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate failing query", func(t *testing.T) {
		query, err := ParseQuery("documents.read AND documents.delete")
		require.NoError(t, err)

		userPermissions := []string{"documents.read"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.False(t, result.Valid)
		require.Contains(t, result.Message, "Missing permission: 'documents.delete'")
	})

	t.Run("Parse and evaluate OR query", func(t *testing.T) {
		query, err := ParseQuery("documents.read OR documents.write")
		require.NoError(t, err)

		userPermissions := []string{"documents.write"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate portal query", func(t *testing.T) {
		query, err := ParseQuery("portal.session.read AND portal.session.create")
		require.NoError(t, err)

		userPermissions := []string{
			"portal.session.read",
			"portal.session.create",
		}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.True(t, result.Valid)
	})

	t.Run("Parse and evaluate failing portal query", func(t *testing.T) {
		query, err := ParseQuery("portal.session.create")
		require.NoError(t, err)

		// Portal management does not imply session minting.
		userPermissions := []string{"portal.session.read"}
		result, err := rbac.EvaluatePermissions(query, userPermissions)
		require.NoError(t, err)
		require.False(t, result.Valid)
		require.Contains(t, result.Message, "Missing permission: 'portal.session.create'")
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
