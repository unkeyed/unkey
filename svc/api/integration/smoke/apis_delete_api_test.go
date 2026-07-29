package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeleteAPI_DeletesAPI(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	created, err := client.Apis.CreateAPI(ctx, components.V2ApisCreateAPIRequestBody{Name: uid.DNS1035()})
	require.NoError(t, err)
	require.NotNil(t, created.V2ApisCreateAPIResponseBody)
	deleted, err := client.Apis.DeleteAPI(ctx, components.V2ApisDeleteAPIRequestBody{APIID: created.V2ApisCreateAPIResponseBody.Data.APIID})
	require.NoError(t, err)
	require.NotNil(t, deleted.V2ApisDeleteAPIResponseBody)
}
