package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliRevokeSessionRequestBody
type Response = openapi.V2CliRevokeSessionResponseBody

type Handler struct {
	Grants *grant.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.revokeSession" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	if err := grant.RequireUserSession(p); err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if err := h.Grants.Revoke(ctx, grant.RevokeRequest{
		Principal: p,
		SessionID: req.SessionId,
		RemoteIP:  s.Location(),
		UserAgent: s.UserAgent(),
	}); err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
