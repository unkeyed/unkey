package vault

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func TestLiveness_NoAuthRequired(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.LivenessRequest{})

	res, err := service.Liveness(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "ok", res.Msg.GetStatus())
}
