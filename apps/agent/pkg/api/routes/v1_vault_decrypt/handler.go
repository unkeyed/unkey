package v1VaultDecrypt

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/vault.v1.VaultService/Decrypt", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		req := &openapi.V1DecryptRequestBody{}
		err := svc.OpenApiValidator.Body(c, req)
		if err != nil {
			return err
		}
		res, err := svc.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   req.Keyring,
			Encrypted: req.Encrypted,
		})
		if err != nil {
			return fault.Wrap(err, fmsg.With("failed to decrypt"))
		}

		return c.JSON(openapi.V1DecryptResponseBody{
			Plaintext: res.Plaintext,
		})
	})
}
