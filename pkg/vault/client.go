package vault

import (
	"context"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

// Client defines the interface for vault encryption and decryption operations.
// Both the local [Service] and the remote [ConnectClient] implement this interface.
type Client interface {
	Encrypt(ctx context.Context, req *vaultv1.EncryptRequest) (*vaultv1.EncryptResponse, error)
	Decrypt(ctx context.Context, req *vaultv1.DecryptRequest) (*vaultv1.DecryptResponse, error)
}

// Compile-time check that *Service implements Client.
var _ Client = (*Service)(nil)
