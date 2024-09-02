package v1VaultEncryptBulk

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

type Request = openapi.V1EncryptBulkRequestBody
type Response = openapi.V1EncryptBulkResponseBody

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/vault.v1.VaultService/EncryptBulk", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		req := Request{}
		err := c.BodyParser(&req)
		if err != nil {
			return fault.Wrap(err)
		}
		res, err := svc.Vault.EncryptBulk(ctx, &vaultv1.EncryptBulkRequest{
			Keyring: req.Keyring,
			Data:    req.Data,
		})
		if err != nil {
			return fault.Wrap(err, fmsg.With("failed to encrypt"))
		}

		encrypted := make([]openapi.Encrypted, len(res.Encrypted))
		for i, e := range res.Encrypted {
			encrypted[i] = openapi.Encrypted{
				Encrypted: e.Encrypted,
				KeyId:     e.KeyId,
			}
		}

		return c.JSON(Response{Encrypted: encrypted})
	})
}
