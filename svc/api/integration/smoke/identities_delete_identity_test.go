package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeleteIdentity_DeletesIdentity(t *testing.T) {
	ctx, client := externalClient(t)
	created, err := client.Identities.CreateIdentity(ctx, components.V2IdentitiesCreateIdentityRequestBody{ExternalID: uid.DNS1035()})
	require.NoError(t, err)
	require.NotNil(t, created.V2IdentitiesCreateIdentityResponseBody)
	waitForPropagation()
	deleted, err := client.Identities.DeleteIdentity(ctx, components.V2IdentitiesDeleteIdentityRequestBody{Identity: created.V2IdentitiesCreateIdentityResponseBody.Data.IdentityID})
	require.NoError(t, err)
	require.NotNil(t, deleted.V2IdentitiesDeleteIdentityResponseBody)
}
