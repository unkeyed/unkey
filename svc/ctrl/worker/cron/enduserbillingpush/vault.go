package enduserbillingpush

import (
	"context"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/fault"
)

type vaultDecrypter struct {
	client vault.VaultServiceClient
}

var _ Decrypter = (*vaultDecrypter)(nil)

// NewVaultDecrypter adapts the vault RPC client to the Decrypter interface.
func NewVaultDecrypter(client vault.VaultServiceClient) Decrypter {
	return &vaultDecrypter{client: client}
}

func (v *vaultDecrypter) Decrypt(ctx context.Context, keyring, encrypted string) (string, error) {
	res, err := v.client.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encrypted,
	})
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("vault decrypt failed"))
	}
	return res.GetPlaintext(), nil
}
