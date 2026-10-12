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
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	view, err := h.Devices.Get(ctx, p, req.UserCode)
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliGetDeviceLoginResponseData{
			UserCode:           view.UserCode,
			DeviceName:         optional(view.DeviceName),
			RequesterIp:        optional(view.RequesterIP),
			RequesterUserAgent: optional(view.RequesterUserAgent),
			Status:             view.Status,
			CreatedAt:          view.CreatedAt,
			ExpiresAt:          view.ExpiresAt,
		},
	})
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
