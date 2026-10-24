package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliDecideDeviceLoginRequestBody
type Response = openapi.V2CliDecideDeviceLoginResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.decideDeviceLogin" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	if err := clidevice.RequireDashboardSession(p); err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	switch req.Decision {
	case openapi.V2CliDecideDeviceLoginRequestBodyDecisionApprove:
		if req.Permissions == nil || len(*req.Permissions) == 0 {
			return fault.New("cli login needs a permission",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Select at least one permission."),
			)
		}
		validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, p, *req.Permissions)
		if err != nil {
			return err
		}
		if err := h.Devices.Approve(ctx, clidevice.ApproveRequest{
			Principal:   p,
			UserCode:    req.UserCode,
			Permissions: validated,
		}); err != nil {
			return err
		}
	case openapi.V2CliDecideDeviceLoginRequestBodyDecisionDeny:
		if req.Permissions != nil {
			return fault.New("cli denial cannot grant permissions",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Omit permissions when denying a login."),
			)
		}
		if err := h.Devices.Deny(ctx, clidevice.DenyRequest{
			Principal: p,
			UserCode:  req.UserCode,
		}); err != nil {
			return err
		}
	default:
		return fault.New("invalid cli login decision",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Choose approve or deny."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
