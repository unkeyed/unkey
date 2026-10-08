package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	query := S("documents.read")

	require.NoError(t, Check(query, []string{"documents.read"}))
	require.Error(t, Check(query, []string{"documents.write"}))
}
