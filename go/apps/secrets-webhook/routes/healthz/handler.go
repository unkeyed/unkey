package healthz

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct{}

func (h *Handler) Method() string {
	return "GET"
}

func (h *Handler) Path() string {
	return "/healthz"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	return s.Plain(http.StatusOK, []byte("ok"))
}
