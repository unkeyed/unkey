package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestUpdateIdentity_PersistsMetadata(t *testing.T) {
	ctx, client := externalClient(t)
	identity := createIdentity(t, ctx, client)
	meta := map[string]any{"smokeTest": uid.DNS1035()}
	response, err := client.Identities.UpdateIdentity(ctx, components.V2IdentitiesUpdateIdentityRequestBody{Identity: identity.ID, Meta: meta})
	require.NoError(t, err)
	require.NotNil(t, response.V2IdentitiesUpdateIdentityResponseBody)
	get, err := client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{Identity: identity.ID})
	require.NoError(t, err)
	require.NotNil(t, get.V2IdentitiesGetIdentityResponseBody)
	require.Equal(t, meta, get.V2IdentitiesGetIdentityResponseBody.Data.Meta)
}
