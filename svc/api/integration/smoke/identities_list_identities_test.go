package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestListIdentities_ReturnsCreatedIdentity(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	createIdentity(t, ctx, client)
	response, err := client.Identities.ListIdentities(ctx, components.V2IdentitiesListIdentitiesRequestBody{Limit: ptr.P(int64(10))})
	require.NoError(t, err)
	require.NotNil(t, response.V2IdentitiesListIdentitiesResponseBody)
	require.NotEmpty(t, response.V2IdentitiesListIdentitiesResponseBody.Data)
}
