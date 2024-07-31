package v1VaultEncryptBulk

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type v1EncryptBulkRequest struct {
	Body struct {
		Keyring string   `json:"keyring" required:"true"`
		Data    []string `json:"data" required:"true" minItems:"1" maxItems:"1000"`
	}
}

type encrypted struct {
	Encrypted string `json:"encrypted" required:"true"`
	KeyID     string `json:"keyId" required:"true"`
}

type v1EncryptBulkResponse struct {
	Body struct {
		Encrypted []encrypted `json:"encrypted"`
	}
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"vault"},
		OperationID: "vault.v1.encryptBulk",
		Method:      "POST",
		Path:        "/vault.v1.VaultService/EncryptBulk",
		Middlewares: middlewares,
	}, func(ctx context.Context, req *v1EncryptBulkRequest) (*v1EncryptBulkResponse, error) {

		ctx, span := tracing.Start(ctx, tracing.NewSpanName("vault", "EncryptBulk"))
		defer span.End()

		res, err := svc.Vault.EncryptBulk(ctx, &vaultv1.EncryptBulkRequest{
			Keyring: req.Body.Keyring,
			Data:    req.Body.Data,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("unable to encrypt", err)
		}

		response := v1EncryptBulkResponse{}

		response.Body.Encrypted = make([]encrypted, len(res.Encrypted))
		for i, e := range res.Encrypted {
			response.Body.Encrypted[i] = encrypted{
				Encrypted: e.Encrypted,
				KeyID:     e.KeyId,
			}
		}

		return &response, nil
	})
}
