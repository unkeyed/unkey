package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliPollDeviceLoginRequestBody
type Response = openapi.V2CliPollDeviceLoginResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.pollDeviceLogin" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	result, err := h.Devices.Poll(ctx, req.LoginId, s.Location(), s.UserAgent())
	if err != nil {
		return err
	}
	data := openapi.V2CliPollDeviceLoginResponseData{
		Status:   openapi.V2CliPollDeviceLoginResponseDataStatus(result.Status),
		Interval: int64(result.Interval),
		Detail:   nil,
		Key:      nil,
		KeyId:    nil,
	}
	if result.Detail != "" {
		data.Detail = &result.Detail
	}
	if result.Key != nil {
		data.Key = result.Key
		data.KeyId = &result.KeyID
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
