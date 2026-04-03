package deployment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureDeploymentSecret_InvalidKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		envKey string
	}{
		{name: "space in key", envKey: "invalid key"},
		{name: "equals sign", envKey: "KEY=VALUE"},
		{name: "sentence as key", envKey: "when I click the i below it submits"},
		{name: "slash", envKey: "path/to/key"},
		{name: "at sign", envKey: "my@key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Controller{clientSet: fake.NewSimpleClientset()}
			err := c.ensureDeploymentSecret(context.Background(), "default", "dep-abc",
				map[string]string{tt.envKey: "value"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "contains invalid characters")
		})
	}
}

func TestEnsureDeploymentSecret_ValidKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		envKey string
	}{
		{name: "uppercase", envKey: "MY_VAR"},
		{name: "lowercase", envKey: "my_var"},
		{name: "with dot", envKey: "app.config"},
		{name: "with dash", envKey: "my-key"},
		{name: "alphanumeric", envKey: "KEY123"},
		{name: "mixed", envKey: "My-App.Config_VAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Controller{clientSet: fake.NewSimpleClientset()}
			err := c.ensureDeploymentSecret(context.Background(), "default", "dep-abc",
				map[string]string{tt.envKey: "value"})
			require.NoError(t, err)
		})
	}
}
