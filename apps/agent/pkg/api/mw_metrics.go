package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
)

func createMetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		serviceLatency := time.Since(start)
		prometheus.HTTPRequests.With(map[string]string{
			"method": c.Method(),
			"path":   c.Path(),
			"status": fmt.Sprintf("%d", c.Response().StatusCode()),
		}).Inc()

		prometheus.ServiceLatency.WithLabelValues(c.Path()).Observe(serviceLatency.Seconds())
		return err
	}

}
