package v1VaultEncryptBulk

import (
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
)

type Request = openapi.V1EncryptBulkRequestBody
type Response = openapi.V1EncryptBulkResponseBody

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/vault.v1.VaultService/EncryptBulk", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req := Request{}
		errorResponse, valid := svc.OpenApiValidator.Body(r, &req)
		if !valid {
			svc.Sender.Send(ctx, w, 400, errorResponse)
			return
		}
		res, err := svc.Vault.EncryptBulk(ctx, &vaultv1.EncryptBulkRequest{
			Keyring: req.Keyring,
			Data:    req.Data,
		})
		if err != nil {
			errors.HandleError(ctx, fault.Wrap(err, fmsg.With("failed to encrypt")))
			return
		}

		encrypted := make([]openapi.Encrypted, len(res.Encrypted))
		for i, e := range res.Encrypted {
			encrypted[i] = openapi.Encrypted{
				Encrypted: e.Encrypted,
				KeyId:     e.KeyId,
			}
		}

		svc.Sender.Send(ctx, w, 200, Response{Encrypted: encrypted})
	})
}
