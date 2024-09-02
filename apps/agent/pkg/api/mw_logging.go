package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

func createLoggerMiddleware(logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		logger.Info().Str("serviceLatency", time.Since(start).String()).Str("method", c.Method()).Str("path", c.Path()).Int("status", c.Response().StatusCode()).Msg("request")
		return err
	}
}
