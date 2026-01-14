package vault

import (
	"context"
	"encoding/base64"
	"fmt"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Decrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.DecryptRequest],
) (*connect.Response[vaultv1.DecryptResponse], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	res, err := s.decrypt(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(res), nil

}

func (s *Service) decrypt(
	ctx context.Context,
	req *vaultv1.DecryptRequest,
) (*vaultv1.DecryptResponse, error) {
	ctx, span := tracing.Start(ctx, "vault.Decrypt")
	defer span.End()

	b, err := base64.StdEncoding.DecodeString(req.GetEncrypted())
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}
	encrypted := vaultv1.Encrypted{} // nolint:exhaustruct
	err = proto.Unmarshal(b, &encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	// Validate the encrypted message structure
	if err := validateEncrypted(&encrypted); err != nil {
		return nil, fmt.Errorf("invalid encrypted message: %w", err)
	}

	cacheKey := fmt.Sprintf("%s-%s", req.GetKeyring(), encrypted.GetEncryptionKeyId())

	dek, hit := s.keyCache.Get(ctx, cacheKey)
	if hit == cache.Miss {
		dek, err = s.keyring.GetKey(ctx, req.GetKeyring(), encrypted.GetEncryptionKeyId())
		if err != nil {
			return nil, fmt.Errorf("failed to get dek in keyring %s: %w", req.GetKeyring(), err)
		}
		s.keyCache.Set(ctx, cacheKey, dek)
	}

	plaintext, err := encryption.Decrypt(dek.GetKey(), encrypted.GetNonce(), encrypted.GetCiphertext())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return &vaultv1.DecryptResponse{
		Plaintext: string(plaintext),
	}, nil

}

// validateEncrypted validates the structure of an Encrypted message.
//
// This validation is critical for security:
//   - Nonce must be exactly 12 bytes (GCM requirement)
//   - Ciphertext must be at least 16 bytes (GCM auth tag size)
//   - Encryption key ID must not be empty
//
// Without this validation, malformed messages could cause panics or
// undefined behavior in the crypto library.
//
// Note: Proto validation (buf.validate) on the Encrypted message is NOT
// automatically enforced because we manually unmarshal it. This Go validation
// provides the actual security guarantee.
func validateEncrypted(e *vaultv1.Encrypted) error {
	const (
		gcmNonceSize   = 12
		gcmAuthTagSize = 16
	)

	if err := assert.Equal(len(e.GetNonce()), gcmNonceSize, fmt.Sprintf("invalid nonce length: expected %d bytes, got %d", gcmNonceSize, len(e.GetNonce()))); err != nil {
		return err
	}

	if err := assert.GreaterOrEqual(len(e.GetCiphertext()), gcmAuthTagSize, fmt.Sprintf("invalid ciphertext length: expected at least %d bytes, got %d", gcmAuthTagSize, len(e.GetCiphertext()))); err != nil {
		return err
	}

	if err := assert.NotEmpty(e.GetEncryptionKeyId(), "encryption key ID is required"); err != nil {
		return err
	}

	return nil
}
