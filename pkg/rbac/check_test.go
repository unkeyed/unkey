package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasAnyPermission(t *testing.T) {
	t.Parallel()

	// HasAnyPermission must match by resource type and action without requiring a resource ID.
	granted := []string{
		"api.api_123.verify_key",
		"key.*.read_key",
		"unkey:v1:ws_123:keyspaces/*/keys/*#verify_key",
	}

	require.True(t, HasAnyPermission(granted, Api, VerifyKey))
	require.False(t, HasAnyPermission(granted, Api, CreateAPI))
}

func TestCheck(t *testing.T) {
	t.Parallel()

	// Check must return nil only when the granted permissions satisfy the query.
	query := T(Tuple{
		ResourceType: Api,
		ResourceID:   "*",
		Action:       CreateAPI,
	})

	require.NoError(t, Check(query, []string{"api.*.create_api"}))
	require.Error(t, Check(query, []string{"api.*.verify_key"}))
}
