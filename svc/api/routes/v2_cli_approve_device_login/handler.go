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

type Request = openapi.V2CliApproveDeviceLoginRequestBody
type Response = openapi.V2CliApproveDeviceLoginResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.approveDeviceLogin" }

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

	validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, p, req.Permissions)
	if err != nil {
		return err
	}
	if len(validated) == 0 {
		return fault.New("cli login needs a permission",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Select at least one permission."),
		)
	}

	if err := h.Devices.Approve(ctx, clidevice.ApproveRequest{
		Principal:   p,
		UserCode:    req.UserCode,
		Permissions: validated,
	}); err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
