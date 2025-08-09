package internalVaultDecrypt

import (
	"context"
	"net/http"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
)

type Handler struct {
	Vault  *vault.Service
	Logger logging.Logger
	Token  string
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/_internal/vault/decrypt"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// Authenticate using Bearer token
	err := zen.StaticAuth(s, h.Token)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[openapi.InternalVaultDecryptRequestBody](s)
	if err != nil {
		return err
	}

	res, err := h.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   req.Keyring,
		Encrypted: req.Encrypted,
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to decrypt data"))
	}

	return s.JSON(http.StatusOK, openapi.InternalVaultDecryptResponseBody{
		Plaintext: res.Plaintext,
	})
}