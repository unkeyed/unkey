package urn_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
)

func TestNotificationPermissionsAreWorkspaceScoped(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	resource := urn.New().Workspace(workspaceID).Notifications()
	parsed, err := urn.ParseV1(resource.String())
	require.NoError(t, err)
	require.Equal(t, resource, parsed)
	require.True(t, resource.SupportsPermissionAction(urn.PermissionRead))
	require.True(t, resource.SupportsPermissionAction(urn.PermissionWrite))
	require.False(t, resource.SupportsPermissionAction(urn.PermissionDelete))
	require.False(t, resource.SupportsPermissionAction(urn.PermissionDecrypt))
	for _, action := range []permissions.Action{permissions.Read, permissions.Write} {
		query := rbac.U(resource, action)
		require.NoError(t, rbac.Check(query, []string{resource.String() + "#" + action.String()}))
		require.NoError(t, rbac.Check(query, []string{"unkey:v1:" + workspaceID + ":**#*"}))
		require.Error(t, rbac.Check(query, []string{urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Notifications().String() + "#" + action.String()}))
		require.Error(t, rbac.Check(query, []string{urn.New().Workspace(workspaceID).Flags().String() + "#" + action.String()}))
	}
}
