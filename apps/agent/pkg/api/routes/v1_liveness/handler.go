package v1Liveness

import (
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("GET", "v1/liveness",
		func(c *fiber.Ctx) error {

			svc.Logger.Debug().Msg("incoming liveness check")
			return c.JSON(openapi.V1LivenessResponseBody{
				Message: "OK",
			})
		},
	)
}
