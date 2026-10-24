package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliExchangeTokenRequestBody
type Response = openapi.V2CliExchangeTokenResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.exchangeToken" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.RefreshToken != nil {
		tokens, err := h.Devices.Refresh(ctx, clidevice.RefreshRequest{
			RefreshToken: *req.RefreshToken,
			RemoteIP:     s.Location(),
			UserAgent:    s.UserAgent(),
		})
		if err != nil {
			return err
		}
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.V2CliExchangeTokenResponseData{
				Credentials: &openapi.V2CliExchangeTokenCredentials{
					AccessToken:  tokens.AccessToken,
					RefreshToken: tokens.RefreshToken,
					ExpiresIn:    tokens.ExpiresIn,
					WorkspaceId:  tokens.WorkspaceID,
					SessionId:    tokens.SessionID,
				},
				Detail:   nil,
				Interval: 0,
				Status:   openapi.V2CliExchangeTokenResponseDataStatus(clidevice.PollComplete),
			},
		})
	}

	result, err := h.Devices.Poll(ctx, clidevice.PollRequest{
		LoginID:   *req.LoginId,
		RemoteIP:  s.Location(),
		UserAgent: s.UserAgent(),
	})
	if err != nil {
		return err
	}

	data := openapi.V2CliExchangeTokenResponseData{
		Credentials: nil,
		Detail:      nil,
		Interval:    int64(result.Interval),
		Status:      openapi.V2CliExchangeTokenResponseDataStatus(result.Status),
	}
	if result.Detail != "" {
		data.Detail = &result.Detail
	}
	if result.Status == clidevice.PollComplete {
		data.Credentials = &openapi.V2CliExchangeTokenCredentials{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresIn:    result.ExpiresIn,
			WorkspaceId:  result.WorkspaceID,
			SessionId:    result.SessionID,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
