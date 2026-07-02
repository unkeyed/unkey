package deploy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeK8sName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// Deployment IDs carry an underscore prefix separator.
		{in: "dep_2xKq9mAbCdEf", want: "dep-2xkq9mabcdef"},
		{in: "already-valid-123", want: "already-valid-123"},
		// Leading/trailing invalid chars must not leave stray hyphens.
		{in: "_dep_", want: "dep"},
		{in: "a__b..c", want: "a-b-c"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, sanitizeK8sName(tt.in))
	}
}

func TestImageExportsInsecure(t *testing.T) {
	//nolint:exhaustruct
	secure := &Workflow{registryConfig: RegistryConfig{Repository: "registry.acme.com/deployments"}}
	exports := secure.imageExports("registry.acme.com/deployments:p-d")
	require.Len(t, exports, 1)
	require.Equal(t, "registry.acme.com/deployments:p-d", exports[0].Attrs["name"])
	require.NotContains(t, exports[0].Attrs, "registry.insecure")

	//nolint:exhaustruct
	insecure := &Workflow{registryConfig: RegistryConfig{Repository: "ctlptl-registry:5000/deployments", Insecure: true}}
	exports = insecure.imageExports("ctlptl-registry:5000/deployments:p-d")
	require.Equal(t, "true", exports[0].Attrs["registry.insecure"])
}
