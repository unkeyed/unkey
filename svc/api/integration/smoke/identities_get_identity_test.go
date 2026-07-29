package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestGetIdentity_ReturnsCreatedIdentity(t *testing.T) {
	ctx, client := externalClient(t)
	identity := createIdentity(t, ctx, client)
	response, err := client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{Identity: identity.ID})
	require.NoError(t, err)
	require.NotNil(t, response.V2IdentitiesGetIdentityResponseBody)
	require.Equal(t, identity.ExternalID, response.V2IdentitiesGetIdentityResponseBody.Data.ExternalID)
}
