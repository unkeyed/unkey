package internalCacheInvalidate

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

type Handler struct {
	Caches caches.Caches
	Token  string
}

type request struct {
	CacheName string   `json:"cacheName"`
	Keys      []string `json:"keys"`
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

	req, err := zen.BindBody[request](s)
	if err != nil {
		return err
	}

	if err := h.Caches.Invalidate(ctx, req.CacheName, req.Keys); err != nil {
		return fault.Wrap(err, fault.Public("Failed to invalidate cache"))
	}

	return s.Send(http.StatusOK, nil)
}
