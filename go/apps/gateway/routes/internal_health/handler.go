package handler

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger logging.Logger
}

func (h *Handler) Method() string {
	return "GET"
}

func (h *Handler) Path() string {
	return "/unkey/internal/health"
}

// Handle returns a simple 200 OK response for health checks
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	return sess.Plain(200, []byte("OK"))
}
