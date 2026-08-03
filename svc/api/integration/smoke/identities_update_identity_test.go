package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestUpdateIdentity_PersistsMetadata(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	identity := createIdentity(t, ctx, client)
	meta := map[string]any{"smokeTest": uid.DNS1035()}
	response, err := client.Identities.UpdateIdentity(ctx, components.V2IdentitiesUpdateIdentityRequestBody{Identity: identity.ID, Meta: meta})
	require.NoError(t, err)
	require.NotNil(t, response.V2IdentitiesUpdateIdentityResponseBody)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, getErr := client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{Identity: identity.ID})
		require.NoError(c, getErr)
		require.NotNil(c, get.V2IdentitiesGetIdentityResponseBody)
		require.Equal(c, meta, get.V2IdentitiesGetIdentityResponseBody.Data.Meta)
	}, 30*time.Second, time.Second)
}
