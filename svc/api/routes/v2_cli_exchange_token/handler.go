package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
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

	if (req.LoginId == nil) == (req.RefreshToken == nil) {
		return fault.New("cli token exchange needs exactly one credential",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Provide exactly one of loginId or refreshToken."),
		)
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
				AccessToken:  &tokens.AccessToken,
				Detail:       nil,
				ExpiresIn:    &tokens.ExpiresIn,
				Interval:     0,
				RefreshToken: &tokens.RefreshToken,
				SessionId:    &tokens.SessionID,
				Status:       openapi.V2CliExchangeTokenResponseDataStatus(clidevice.PollComplete),
				WorkspaceId:  &tokens.WorkspaceID,
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
		AccessToken:  nil,
		Detail:       nil,
		ExpiresIn:    nil,
		Interval:     int64(result.Interval),
		RefreshToken: nil,
		SessionId:    nil,
		Status:       openapi.V2CliExchangeTokenResponseDataStatus(result.Status),
		WorkspaceId:  nil,
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
