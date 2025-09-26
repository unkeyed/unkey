package v2Liveness

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

type Response = openapi.V2LivenessResponseBody

// Handler implements zen.Route interface for the v2 liveness endpoint
type Handler struct {
	// No services needed for liveness check
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "GET"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/liveness"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.DisableClickHouseLogging()

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2LivenessResponseData{
			Message: "we're cooking",
		},
	})
}
