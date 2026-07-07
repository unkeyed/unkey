package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
