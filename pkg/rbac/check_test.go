package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestCheck_Allows_WhenGrantedSatisfiesQuery(t *testing.T) {
	query := T(Tuple{ResourceType: Api, ResourceID: "api_1", Action: ReadAPI})

	err := Check(query, []string{"api.api_1.read_api"})

	require.NoError(t, err)
}

func TestCheck_Denies_WithInsufficientPermissionsCode(t *testing.T) {
	query := T(Tuple{ResourceType: Api, ResourceID: "api_1", Action: ReadAPI})

	err := Check(query, []string{"api.api_other.read_api"})

	require.Error(t, err)
	gotCode, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), gotCode)
}

func TestCheck_Denies_OrQuery_WhenNoBranchMatches(t *testing.T) {
	query := Or(
		T(Tuple{ResourceType: Api, ResourceID: "*", Action: ReadAPI}),
		T(Tuple{ResourceType: Api, ResourceID: "api_1", Action: ReadAPI}),
	)

	err := Check(query, []string{"ratelimit.ns_1.limit"})

	require.Error(t, err)
	gotCode, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), gotCode)
}

func TestCheck_Allows_OrQuery_WhenAnyBranchMatches(t *testing.T) {
	query := Or(
		T(Tuple{ResourceType: Api, ResourceID: "*", Action: ReadAPI}),
		T(Tuple{ResourceType: Api, ResourceID: "api_1", Action: ReadAPI}),
	)

	err := Check(query, []string{"api.api_1.read_api"})

	require.NoError(t, err)
}
