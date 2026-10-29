package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
)

// Handler serves RFC 8414 authorization server metadata. The issuer has no
// path, so the well-known path takes no suffix.
type Handler struct {
	Metadata oauth.Metadata
}

func (h *Handler) Method() string { return http.MethodGet }

func (h *Handler) Path() string { return oauth.MetadataPath }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.AddHeader("Cache-Control", "public, max-age="+strconv.Itoa(int(oauth.MetadataMaxAge.Seconds())))
	return s.JSON(http.StatusOK, h.Metadata)
}
