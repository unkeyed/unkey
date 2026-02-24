package internalCacheInvalidate

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.CacheInvalidateRequestBody
	Response = struct{}
)

type Handler struct {
	Caches caches.Caches
	Token  string
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/_internal/cache.invalidate"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.DisableClickHouseLogging()

	if err := zen.BearerTokenAuth(s, h.Token); err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if err := h.Caches.Invalidate(ctx, req.CacheName, req.Keys); err != nil {
		return fault.Wrap(err, fault.Public("Failed to invalidate cache"))
	}

	return s.JSON(http.StatusOK, Response{})
}
