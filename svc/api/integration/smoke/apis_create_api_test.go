package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreateAPI_ReturnsAPIID(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	name := uid.DNS1035()
	response, err := client.Apis.CreateAPI(ctx, components.V2ApisCreateAPIRequestBody{Name: name})
	require.NoError(t, err)
	require.NotNil(t, response.V2ApisCreateAPIResponseBody)
	require.NotEmpty(t, response.V2ApisCreateAPIResponseBody.Data.APIID)
	t.Cleanup(func() {
		_, err := client.Apis.DeleteAPI(ctx, components.V2ApisDeleteAPIRequestBody{APIID: response.V2ApisCreateAPIResponseBody.Data.APIID})
		require.NoError(t, err)
	})
}
