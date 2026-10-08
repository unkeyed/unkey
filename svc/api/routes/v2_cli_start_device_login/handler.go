package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliStartDeviceLoginRequestBody
type Response = openapi.V2CliStartDeviceLoginResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.startDeviceLogin" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	deviceName := ""
	if req.DeviceName != nil {
		deviceName = *req.DeviceName
	}
	started, err := h.Devices.Start(ctx, clidevice.StartRequest{
		DeviceName: deviceName,
		RemoteIP:   s.Location(),
		UserAgent:  s.UserAgent(),
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliStartDeviceLoginResponseData{
			LoginId:                 started.LoginID,
			UserCode:                started.UserCode,
			VerificationUri:         started.VerificationURI,
			VerificationUriComplete: started.VerificationURIComplete,
			ExpiresIn:               int64(started.ExpiresIn),
			Interval:                int64(started.Interval),
		},
	})
}
