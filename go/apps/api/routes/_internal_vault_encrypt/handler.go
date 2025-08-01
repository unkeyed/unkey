package internalVaultEncrypt

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
	return "/_internal/vault/encrypt"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// Authenticate using Bearer token
	err := zen.StaticAuth(s, h.Token)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[openapi.InternalVaultEncryptRequestBody](s)
	if err != nil {
		return err
	}

	res, err := h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: req.Keyring,
		Data:    req.Data,
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to encrypt data"))
	}

	return s.JSON(http.StatusOK, openapi.InternalVaultEncryptResponseBody{
		Encrypted: res.Encrypted,
		KeyId:     res.KeyId,
	})
}