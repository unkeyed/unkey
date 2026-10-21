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

	result, err := h.Devices.Poll(ctx, clidevice.PollRequest{
		LoginID:   req.LoginId,
		RemoteIP:  s.Location(),
		UserAgent: s.UserAgent(),
	})
	if err != nil {
		return err
	}

	data := openapi.V2CliPollDeviceLoginResponseData{
		Status:   openapi.V2CliPollDeviceLoginResponseDataStatus(result.Status),
		Interval: int64(result.Interval),
	}
	if result.Detail != "" {
		data.Detail = &result.Detail
	}
	if result.AccessToken != "" {
		data.AccessToken = &result.AccessToken
		data.RefreshToken = &result.RefreshToken
		expiresIn := result.ExpiresIn
		data.ExpiresIn = &expiresIn
		data.WorkspaceId = &result.WorkspaceID
		data.SessionId = &result.SessionID
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
