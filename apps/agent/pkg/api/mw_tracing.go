package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func tracingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		ctx, span := tracing.Start(ctx, tracing.NewSpanName("api", c.Path()))
		defer span.End()
		c.SetUserContext(ctx)
		return c.Next()
	}
}
