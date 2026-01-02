package v1VaultEncrypt

import (
	"net/http"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/vault.v1.VaultService/Encrypt",
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			req := &openapi.V1EncryptRequestBody{}
			errorResponse, valid := svc.OpenApiValidator.Body(r, req)
			if !valid {
				svc.Sender.Send(ctx, w, 400, errorResponse)
				return
			}
			res, err := svc.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
				Keyring: req.Keyring,
				Data:    req.Data,
			})
			if err != nil {
				errors.HandleError(ctx, err)
				return
			}

			svc.Sender.Send(ctx, w, 200, openapi.V1EncryptResponseBody{
				Encrypted: res.Encrypted,
				KeyId:     res.KeyId,
			})
		})
}
