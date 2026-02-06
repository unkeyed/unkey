package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
)

type Handler struct{}

func (h *Handler) Method() string {
	return "GET"
}

func (h *Handler) Path() string {
	return "/_unkey/internal/health"
}

// Handle returns a simple 200 OK response for health checks
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	sess.DisableClickHouseLogging()
	return sess.Plain(200, []byte("OK"))
}
