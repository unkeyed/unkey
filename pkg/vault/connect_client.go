package vault

import (
	"context"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
)

// Compile-time check that *ConnectClient implements Client.
var _ Client = (*ConnectClient)(nil)

// ConnectClient adapts a [vaultv1connect.VaultServiceClient] to the [Client] interface,
// wrapping and unwrapping connect.Request/Response types.
type ConnectClient struct {
	inner vaultv1connect.VaultServiceClient
}

// NewConnectClient creates a new [ConnectClient] wrapping the given connect client.
func NewConnectClient(inner vaultv1connect.VaultServiceClient) *ConnectClient {
	return &ConnectClient{inner: inner}
}

func (c *ConnectClient) Encrypt(ctx context.Context, req *vaultv1.EncryptRequest) (*vaultv1.EncryptResponse, error) {
	resp, err := c.inner.Encrypt(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *ConnectClient) Decrypt(ctx context.Context, req *vaultv1.DecryptRequest) (*vaultv1.DecryptResponse, error) {
	resp, err := c.inner.Decrypt(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
