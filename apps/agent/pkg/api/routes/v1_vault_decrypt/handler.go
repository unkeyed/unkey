package v1VaultDecrypt

import (
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/vault.v1.VaultService/Decrypt", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req := &openapi.V1DecryptRequestBody{}
		errorResponse, valid := svc.OpenApiValidator.Body(r, req)
		if !valid {
			svc.Sender.Send(ctx, w, 400, errorResponse)
			return
		}
		res, err := svc.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   req.Keyring,
			Encrypted: req.Encrypted,
		})
		if err != nil {
			errors.HandleError(ctx, fault.Wrap(err, fmsg.With("failed to decrypt")))
		}

		svc.Sender.Send(ctx, w, 200, openapi.V1DecryptResponseBody{
			Plaintext: res.Plaintext,
		})
	})
}
