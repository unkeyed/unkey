package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestGetAPI_ReturnsCreatedAPI(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	response, err := client.Apis.GetAPI(ctx, components.V2ApisGetAPIRequestBody{APIID: api.APIID})
	require.NoError(t, err)
	require.NotNil(t, response.V2ApisGetAPIResponseBody)
	require.Equal(t, api.APIID, response.V2ApisGetAPIResponseBody.Data.ID)
}
