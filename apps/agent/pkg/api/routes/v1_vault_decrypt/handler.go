package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type v1DecryptRequest struct {
	Body struct {
		Keyring   string `json:"keyring" required:"true" doc:"The keyring to use for encryption."`
		Encrypted string `json:"encrypted" required:"true" minLength:"1" doc:"The encrypted base64 string."`
	}
}

type v1DecryptResponse struct {
	Body struct {
		Plaintext string `json:"plaintext" required:"true" doc:"The plaintext value."`
	}
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"vault"},
		OperationID: "vault.v1.decrypt",
		Method:      "POST",
		Path:        "/vault.v1.VaultService/Decrypt",
		Middlewares: middlewares,
	}, func(ctx context.Context, req *v1DecryptRequest) (*v1DecryptResponse, error) {

		ctx, span := tracing.Start(ctx, tracing.NewSpanName("vault", "Decrypt"))
		defer span.End()

		res, err := svc.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   req.Body.Keyring,
			Encrypted: req.Body.Encrypted,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("unable to decrypt", err)
		}

		response := v1DecryptResponse{}
		response.Body.Plaintext = res.Plaintext

		return &response, nil
	})
}
