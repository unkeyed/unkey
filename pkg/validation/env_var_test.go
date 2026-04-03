package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidEnvVarKey_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		envKey string
	}{
		{name: "empty", envKey: ""},
		{name: "space in key", envKey: "invalid key"},
		{name: "equals sign", envKey: "KEY=VALUE"},
		{name: "sentence as key", envKey: "when I click the i below it submits"},
		{name: "slash", envKey: "path/to/key"},
		{name: "at sign", envKey: "my@key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.False(t, IsValidEnvVarKey(tt.envKey))
		})
	}
}

func TestIsValidEnvVarKey_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		envKey string
	}{
		{name: "uppercase", envKey: "MY_VAR"},
		{name: "lowercase", envKey: "my_var"},
		{name: "alphanumeric", envKey: "KEY123"},
		{name: "starts with digit", envKey: "1VAR"},
		{name: "with dot", envKey: "app.config"},
		{name: "with hyphen", envKey: "my-key"},
		{name: "mixed", envKey: "My-App.Config_VAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.True(t, IsValidEnvVarKey(tt.envKey))
		})
	}
}
