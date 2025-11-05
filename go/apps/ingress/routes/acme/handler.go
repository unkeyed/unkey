package handler

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger logging.Logger
}

// Handle processes ACME HTTP-01 challenges for Let's Encrypt certificate issuance
// TODO: Implement ACME challenge handler
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	// For now, return 404 - will implement ACME challenge handling later
	return sess.Plain(404, []byte("ACME challenge handler not yet implemented"))
}
