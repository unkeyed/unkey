package portal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestGetSession_EmptyToken_ReturnsError(t *testing.T) {
	t.Parallel()

	svc := &service{}
	ctx := context.Background()

	info, err := svc.GetSession(ctx, "")

	require.Error(t, err)
	require.Nil(t, info)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Portal.Session.TokenMissing.URN(), code)
}

func TestSessionInfo_FieldsExist(t *testing.T) {
	t.Parallel()

	info := SessionInfo{
		SessionID:   "ps_001",
		WorkspaceID: "ws_123",
		ExternalID:  "user_456",
		PortalID:    "pc_789",
		Scopes:      []string{"keys:read", "analytics:read"},
		Preview:     true,
	}

	require.Equal(t, "ps_001", info.SessionID)
	require.Equal(t, "ws_123", info.WorkspaceID)
	require.Equal(t, "user_456", info.ExternalID)
	require.Equal(t, "pc_789", info.PortalID)
	require.Equal(t, []string{"keys:read", "analytics:read"}, info.Scopes)
	require.True(t, info.Preview)
}

func TestSessionInfo_NilScopes(t *testing.T) {
	t.Parallel()

	info := SessionInfo{
		WorkspaceID: "ws_123",
		ExternalID:  "user_456",
		PortalID:    "pc_789",
	}

	require.Nil(t, info.Scopes)
	require.False(t, info.Preview)
}
