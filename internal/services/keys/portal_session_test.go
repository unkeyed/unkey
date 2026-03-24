package keys

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestGetPortalSession_EmptyToken_ReturnsError(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	info, err := s.GetPortalSession(ctx, nil, "")

	require.Error(t, err)
	require.Nil(t, info)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Missing.URN(), code)
}

func TestPortalSessionInfo_FieldsExist(t *testing.T) {
	t.Parallel()

	info := PortalSessionInfo{
		WorkspaceID:    "ws_123",
		ExternalID:     "user_456",
		PortalConfigID: "pc_789",
		Permissions:    []string{"keys:read", "analytics:read"},
		Metadata:       map[string]any{"name": "Test User", "email": "test@example.com"},
		Preview:        true,
	}

	require.Equal(t, "ws_123", info.WorkspaceID)
	require.Equal(t, "user_456", info.ExternalID)
	require.Equal(t, "pc_789", info.PortalConfigID)
	require.Equal(t, []string{"keys:read", "analytics:read"}, info.Permissions)
	require.Equal(t, map[string]any{"name": "Test User", "email": "test@example.com"}, info.Metadata)
	require.True(t, info.Preview)
}

func TestPortalSessionInfo_NilPermissionsAndMetadata(t *testing.T) {
	t.Parallel()

	info := PortalSessionInfo{
		WorkspaceID:    "ws_123",
		ExternalID:     "user_456",
		PortalConfigID: "pc_789",
	}

	require.Nil(t, info.Permissions)
	require.Nil(t, info.Metadata)
	require.False(t, info.Preview)
}
