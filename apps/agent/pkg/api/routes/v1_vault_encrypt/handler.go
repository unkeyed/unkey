package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type v1EncryptRequest struct {
	Body struct {
		Keyring string `json:"keyring" required:"true" doc:"The keyring to use for encryption."`
		Data    string `json:"data" required:"true" minLength:"1" doc:"The data to encrypt."`
	}
}

type v1EncryptResponse struct {
	Body struct {
		Encrypted string `json:"encrypted" required:"true" doc:"The encrypted data as base64 encoded string."`
		KeyID     string `json:"keyId" required:"true" doc:"The ID of the key used for encryption."`
	}
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"vault"},
		OperationID: "vault.v1.encrypt",
		Method:      "POST",
		Path:        "/vault.v1.VaultService/Encrypt",
		Middlewares: middlewares,
	}, func(ctx context.Context, req *v1EncryptRequest) (*v1EncryptResponse, error) {

		ctx, span := tracing.Start(ctx, tracing.NewSpanName("vault", "Encrypt"))
		defer span.End()

		res, err := svc.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: req.Body.Keyring,
			Data:    req.Body.Data,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("unable to encrypt", err)
		}

		response := v1EncryptResponse{}
		response.Body.Encrypted = res.Encrypted
		response.Body.KeyID = res.KeyId

		return &response, nil
	})
}
