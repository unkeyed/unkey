package vault

import (
	"context"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

// Client defines the interface for vault encryption and decryption operations.
// [ConnectClient] implements this interface by wrapping a remote vault service.
type Client interface {
	Encrypt(ctx context.Context, req *vaultv1.EncryptRequest) (*vaultv1.EncryptResponse, error)
	Decrypt(ctx context.Context, req *vaultv1.DecryptRequest) (*vaultv1.DecryptResponse, error)
}
