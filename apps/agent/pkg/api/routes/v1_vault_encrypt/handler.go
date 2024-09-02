package v1VaultEncrypt

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/vault.v1.VaultService/Encrypt",
		func(c *fiber.Ctx) error {
			ctx := c.UserContext()
			req := &openapi.V1EncryptRequestBody{}
			err := svc.OpenApiValidator.Body(c, req)
			if err != nil {
				return err
			}
			res, err := svc.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
				Keyring: req.Keyring,
				Data:    req.Data,
			})
			if err != nil {
				return fault.Wrap(err, fmsg.With("failed to encrypt"))
			}

			return c.JSON(openapi.V1EncryptResponseBody{
				Encrypted: res.Encrypted,
				KeyId:     res.KeyId,
			})
		})
}
