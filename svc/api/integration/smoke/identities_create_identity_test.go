package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreateIdentity_ReturnsIdentityID(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	response, err := client.Identities.CreateIdentity(ctx, components.V2IdentitiesCreateIdentityRequestBody{ExternalID: uid.DNS1035()})
	require.NoError(t, err)
	require.NotNil(t, response.V2IdentitiesCreateIdentityResponseBody)
	require.NotEmpty(t, response.V2IdentitiesCreateIdentityResponseBody.Data.IdentityID)
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Identities.DeleteIdentity(ctx, components.V2IdentitiesDeleteIdentityRequestBody{Identity: response.V2IdentitiesCreateIdentityResponseBody.Data.IdentityID})
		require.NoError(t, err)
	})
}
