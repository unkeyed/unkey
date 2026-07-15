package policies

import (
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
)

func TestSecretLocations(t *testing.T) {
	policies := []*frontlinev1.Policy{
		{
			Config: &frontlinev1.Policy_Keyauth{
				Keyauth: &frontlinev1.KeyAuth{
					Locations: []*frontlinev1.KeyLocation{
						{Location: &frontlinev1.KeyLocation_Bearer{Bearer: &frontlinev1.BearerTokenLocation{}}},
						{Location: &frontlinev1.KeyLocation_Header{Header: &frontlinev1.HeaderKeyLocation{Name: "X-API-Key"}}},
						{Location: &frontlinev1.KeyLocation_QueryParam{QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "api_key"}}},
					},
				},
			},
		},
		// A non-KeyAuth policy is ignored.
		{Config: &frontlinev1.Policy_Firewall{Firewall: &frontlinev1.Firewall{}}},
	}

	headers, queryParams := SecretLocations(policies)

	// Header names are lowercased; Bearer contributes nothing.
	require.Equal(t, []string{"x-api-key"}, headers)
	require.Equal(t, []string{"api_key"}, queryParams)
}

func TestSecretLocations_NoLocations(t *testing.T) {
	policies := []*frontlinev1.Policy{
		{Config: &frontlinev1.Policy_Keyauth{Keyauth: &frontlinev1.KeyAuth{}}},
	}

	headers, queryParams := SecretLocations(policies)

	require.Empty(t, headers)
	require.Empty(t, queryParams)
}

func TestSecretLocations_Empty(t *testing.T) {
	headers, queryParams := SecretLocations(nil)
	require.Empty(t, headers)
	require.Empty(t, queryParams)
}
