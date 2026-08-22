package deploymentgc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryProjectID(t *testing.T) {
	t.Run("extracts project ID", func(t *testing.T) {
		id, err := RegistryProjectID("registry.depot.dev/abc123")
		require.NoError(t, err)
		require.Equal(t, "abc123", id)
	})

	for _, repository := range []string{
		"",
		"example.com/abc123",
		"registry.depot.dev/",
		"registry.depot.dev/team/abc123",
	} {
		t.Run(repository, func(t *testing.T) {
			_, err := RegistryProjectID(repository)
			require.Error(t, err)
		})
	}
}
