package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeletePermission_DeletesPermission(t *testing.T) {
	ctx, client := externalClient(t)
	name := uid.DNS1035()
	created, err := client.Permissions.CreatePermission(ctx, components.V2PermissionsCreatePermissionRequestBody{Name: name, Slug: name})
	require.NoError(t, err)
	require.NotNil(t, created.V2PermissionsCreatePermissionResponseBody)
	waitForPropagation()
	deleted, err := client.Permissions.DeletePermission(ctx, components.V2PermissionsDeletePermissionRequestBody{Permission: created.V2PermissionsCreatePermissionResponseBody.Data.PermissionID})
	require.NoError(t, err)
	require.NotNil(t, deleted.V2PermissionsDeletePermissionResponseBody)
}
