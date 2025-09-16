package handler

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Handler implements zen.Route interface for the API reference endpoint
type Handler struct {
	// Services as public fields (even though not used in this handler, showing the pattern)
	Logger logging.Logger
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "GET"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/openapi.yaml"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.DisableClickHouseLogging()

	s.AddHeader("Content-Type", "application/yaml")
	return s.Send(200, openapi.Spec)
}
