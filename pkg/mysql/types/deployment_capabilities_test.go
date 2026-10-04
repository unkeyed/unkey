package dbtype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentCapabilitiesScanAndValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  DeploymentCapabilities
	}{
		{name: "nil", input: nil, want: DeploymentCapabilities{PrivateNetworking: false}},
		{name: "empty bytes", input: []byte{}, want: DeploymentCapabilities{PrivateNetworking: false}},
		{name: "empty object", input: []byte(`{}`), want: DeploymentCapabilities{PrivateNetworking: false}},
		{name: "enabled", input: []byte(`{"private_networking":true}`), want: DeploymentCapabilities{PrivateNetworking: true}},
		{name: "string input", input: `{"private_networking":true}`, want: DeploymentCapabilities{PrivateNetworking: true}},
		{name: "unknown keys are ignored", input: []byte(`{"private_networking":true,"later":true}`), want: DeploymentCapabilities{PrivateNetworking: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var capabilities DeploymentCapabilities
			require.NoError(t, capabilities.Scan(test.input))
			require.Equal(t, test.want, capabilities)

			value, err := capabilities.Value()
			require.NoError(t, err)
			var roundTrip DeploymentCapabilities
			require.NoError(t, roundTrip.Scan(value))
			require.Equal(t, capabilities, roundTrip)
		})
	}
}

func TestDeploymentCapabilitiesScanResetsPreviousValue(t *testing.T) {
	capabilities := DeploymentCapabilities{PrivateNetworking: true}
	require.NoError(t, capabilities.Scan([]byte(`{}`)))
	require.False(t, capabilities.PrivateNetworking)
}

func TestDeploymentCapabilitiesScanRejectsInvalidInput(t *testing.T) {
	var capabilities DeploymentCapabilities
	require.Error(t, capabilities.Scan(42))
	require.Error(t, capabilities.Scan([]byte(`[]`)))
}
