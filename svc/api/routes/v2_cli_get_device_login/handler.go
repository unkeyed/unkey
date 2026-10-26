package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliGetDeviceLoginRequestBody
type Response = openapi.V2CliGetDeviceLoginResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.getDeviceLogin" }

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

	authz, err := h.Devices.Get(ctx, clidevice.GetRequest{
		Principal: p,
		UserCode:  req.UserCode,
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliGetDeviceLoginResponseData{
			UserCode:           authz.UserCode,
			DeviceName:         optional(authz.DeviceName),
			RequesterIp:        optional(authz.RequesterIP),
			RequesterUserAgent: optional(authz.RequesterUserAgent),
			WorkspaceId:        optional(authz.WorkspaceID),
			Status:             authz.Status,
			CreatedAt:          authz.CreatedAt,
			ExpiresAt:          authz.ExpiresAt,
		},
	})
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
