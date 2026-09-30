package dbtype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentFeaturesScanAndValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  DeploymentFeatures
	}{
		{name: "nil", input: nil, want: DeploymentFeatures{PrivateNetworking: false}},
		{name: "empty bytes", input: []byte{}, want: DeploymentFeatures{PrivateNetworking: false}},
		{name: "empty object", input: []byte(`{}`), want: DeploymentFeatures{PrivateNetworking: false}},
		{name: "enabled", input: []byte(`{"private_networking":true}`), want: DeploymentFeatures{PrivateNetworking: true}},
		{name: "string input", input: `{"private_networking":true}`, want: DeploymentFeatures{PrivateNetworking: true}},
		{name: "unknown keys are ignored", input: []byte(`{"private_networking":true,"later":true}`), want: DeploymentFeatures{PrivateNetworking: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var features DeploymentFeatures
			require.NoError(t, features.Scan(test.input))
			require.Equal(t, test.want, features)

			value, err := features.Value()
			require.NoError(t, err)
			var roundTrip DeploymentFeatures
			require.NoError(t, roundTrip.Scan(value))
			require.Equal(t, features, roundTrip)
		})
	}
}

func TestDeploymentFeaturesScanResetsPreviousValue(t *testing.T) {
	features := DeploymentFeatures{PrivateNetworking: true}
	require.NoError(t, features.Scan([]byte(`{}`)))
	require.False(t, features.PrivateNetworking)
}

func TestDeploymentFeaturesScanRejectsInvalidInput(t *testing.T) {
	var features DeploymentFeatures
	require.Error(t, features.Scan(42))
	require.Error(t, features.Scan([]byte(`[]`)))
}
