package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestListPermissions_ReturnsCreatedPermission(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	createPermission(t, ctx, client)
	response, err := client.Permissions.ListPermissions(ctx, components.V2PermissionsListPermissionsRequestBody{Limit: ptr.P(int64(10))})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsListPermissionsResponseBody)
	require.NotEmpty(t, response.V2PermissionsListPermissionsResponseBody.Data)
}
