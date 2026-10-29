package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliDecideDeviceLoginRequestBody
type Response = openapi.V2CliDecideDeviceLoginResponseBody

type Handler struct {
	Grants *grant.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.decideDeviceLogin" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	if _, err := grant.RequireDashboardSession(p); err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	switch req.Decision {
	case openapi.V2CliDecideDeviceLoginRequestBodyDecisionApprove:
		if err := h.Grants.Approve(ctx, grant.ApproveRequest{
			Principal:   p,
			UserCode:    req.UserCode,
			Permissions: ptr.SafeDeref(req.Permissions),
		}); err != nil {
			return err
		}
	case openapi.V2CliDecideDeviceLoginRequestBodyDecisionDeny:
		if err := h.Grants.Deny(ctx, grant.DenyRequest{
			Principal: p,
			UserCode:  req.UserCode,
		}); err != nil {
			return err
		}
	default:
		return fault.New("unexpected cli login decision")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
